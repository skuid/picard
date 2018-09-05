package metadata

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/skuid/warden/pkg/mapvalue"

	jsoniter "github.com/json-iterator/go"
	"github.com/skuid/picard"
	"github.com/skuid/warden/pkg/ds"

	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/errors"
)

var Deploy = middlewares.Apply(
	http.HandlerFunc(deploy),
	api.AddPicardORMToContext,
)

var Retrieve = middlewares.Apply(
	http.HandlerFunc(retrieve),
	api.AddPicardORMToContext,
)

const (
	datasourceMeta = "datasources"
	profileMeta    = "profiles"
)

var acceptedMetadataTypes = []string{
	datasourceMeta,
	profileMeta,
}

// JsonIter Extension for metadata marshalling/unmarshalling
type namingStrategyExtension struct {
	jsoniter.DummyExtension
}

func (extension *namingStrategyExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	for _, binding := range structDescriptor.Fields {
		metadataTag, hasMetadataTag := binding.Field.Tag().Lookup("metadata-json")
		if hasMetadataTag {
			options := strings.Split(metadataTag, ",")[1:]
			if mapvalue.StringSliceContainsKey(options, "omitretrieve") {
				// Don't do bindings for tags that have the "omitretrieve" option
				binding.ToNames = []string{}
				binding.FromNames = []string{}
			}
		}
	}
}

var deployJSON = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            true,
	ValidateJsonRawMessage: true,
	OnlyTaggedField:        true,
	TagKey:                 "metadata-json",
}.Froze()

func isAcceptableMetadataType(checkType string) bool {
	for _, acceptableType := range acceptedMetadataTypes {
		if checkType == acceptableType {
			return true
		}
	}
	return false
}

func deploy(w http.ResponseWriter, r *http.Request) {
	if isAdmin := api.IsAdminFromContext(r.Context()); !isAdmin {
		api.RespondForbidden(w, errors.ErrUnauthorized)
		return
	}

	picardORM, err := api.PicardORMFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	tmpFileName, err := createTemporaryZipFile(r.Body)
	if err != nil {
		api.RespondInternalError(w, errors.ErrInternal)
		return
	}

	reader, err := zip.OpenReader(tmpFileName)
	if err != nil {
		api.RespondInternalError(w, errors.ErrInternal)
		return
	}

	dataSources := []ds.DataSourceNew{}
	dataSourcePermissions := []ds.DataSourcePermission{}

	// For each file that we encounter, get its root folder plus base name
	for _, file := range reader.File {
		filename := file.Name
		metadataType := filepath.Dir(filename)

		if !isAcceptableMetadataType(metadataType) {
			continue
		}

		fileData, err := file.Open()
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
			return
		}
		defer fileData.Close()
		if metadataType == profileMeta {
			profile := ds.Profile{}
			// Using a custom json decoder so we can use different strut tags for deploy/retrieve
			if err := deployJSON.NewDecoder(fileData).Decode(&profile); err != nil {
				api.RespondInternalError(w, errors.ErrInternal)
				return
			}
			dsPermsToAdd := profile.GetDataSourcePermissions()

			dataSourcePermissions = append(dataSourcePermissions, dsPermsToAdd...)
		} else if metadataType == datasourceMeta {
			dataSource := ds.DataSourceNew{}
			// Using a custom json decoder so we can use different strut tags for deploy/retrieve
			if err := deployJSON.NewDecoder(fileData).Decode(&dataSource); err != nil {
				api.RespondInternalError(w, errors.ErrInternal)
				return
			}
			dataSources = append(dataSources, dataSource)
		}
	}

	// Temporary fix while we still need to handle V1 Data Sources
	// We need to remove any information about V1 Data Sources from the profiles
	// So if the list of data sources does not contain the data source mentioned
	// in the permission set, remove it.
	validDataSourceNames := make([]string, len(dataSources))
	for i, ds := range dataSources {
		validDataSourceNames[i] = ds.Name
	}

	validDataSourcePermissions := []ds.DataSourcePermission{}
	for _, dsPerm := range dataSourcePermissions {
		if mapvalue.StringSliceContainsKey(validDataSourceNames, dsPerm.DataSource.Name) {
			validDataSourcePermissions = append(validDataSourcePermissions, dsPerm)
		}
	}
	// END Temporary FIX to accound for V1 Data sources in profiles

	if err = picardORM.Deploy(dataSources); err != nil {
		api.RespondInternalError(w, err)
		return
	}

	if err = picardORM.Deploy(validDataSourcePermissions); err != nil {
		api.RespondInternalError(w, err)
		return
	}

	resp, err := json.Marshal(map[string]interface{}{})
	if err != nil {
		api.RespondInternalError(w, errors.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func retrieve(w http.ResponseWriter, r *http.Request) {
	// JsonIter Extension for metadata marshalling/unmarshalling
	jsoniter.RegisterExtension(&namingStrategyExtension{jsoniter.DummyExtension{}})

	if isAdmin := api.IsAdminFromContext(r.Context()); !isAdmin {
		api.RespondForbidden(w, errors.ErrUnauthorized)
		return
	}

	picardORM, err := api.PicardORMFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add in Data Sources
	dataSourceResults, err := picardORM.FilterModelAssociations(ds.DataSourceNew{}, []picard.Association{
		{
			Name: "Entities",
			Associations: []picard.Association{
				{
					Name: "Fields",
				},
				{
					Name: "Conditions",
				},
			},
		},
	})
	if err != nil {
		api.RespondInternalError(w, errors.ErrInternal)
		return
	}

	for _, result := range dataSourceResults {
		dataSource := result.(ds.DataSourceNew)
		archivePath := "datasources/" + dataSource.Name + ".json"
		zipFileWriter, err := zipWriter.Create(archivePath)
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
			return
		}

		dsPayload, err := deployJSON.MarshalIndent(&dataSource, "", "   ")
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
			return
		}

		zipFileWriter.Write(dsPayload)
	}

	// Add in Profiles
	dsPermissionResults, err := picardORM.FilterModelAssociations(ds.DataSourcePermission{}, []picard.Association{
		{
			Name: "DataSource",
		},
		{
			Name: "EntityPermissions",
			Associations: []picard.Association{
				{
					Name: "Entity",
				},
				{
					Name: "FieldPermissions",
					Associations: []picard.Association{
						{
							Name: "EntityField",
						},
					},
				},
				{
					Name: "ConditionPermissions",
					Associations: []picard.Association{
						{
							Name: "EntityCondition",
						},
					},
				},
			},
		},
	})

	if err != nil {
		api.RespondInternalError(w, errors.ErrInternal)
		return
	}

	profileMap := map[string]ds.Profile{}

	for _, result := range dsPermissionResults {
		permission := result.(ds.DataSourcePermission)
		profileName := permission.PermissionSetID

		profile, ok := profileMap[profileName]

		if !ok {
			profile = ds.Profile{
				Name: profileName,
				PermissionSet: ds.PermissionSet{
					Name: profileName,
					DataSourcePermissions: map[string]*ds.DataSourcePermission{},
				},
			}
			profileMap[profileName] = profile
		}

		profile.PermissionSet.DataSourcePermissions[permission.DataSource.Name] = &permission
	}

	for _, profile := range profileMap {

		archivePath := "profiles/" + profile.Name + ".json"
		zipFileWriter, err := zipWriter.Create(archivePath)
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
			return
		}

		dsPayload, err := deployJSON.MarshalIndent(&profile, "", "   ")
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
			return
		}

		zipFileWriter.Write(dsPayload)
	}

	err = zipWriter.Close()
	if err != nil {
		api.RespondInternalError(w, errors.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())

}

func createTemporaryZipFile(data io.ReadCloser) (name string, err error) {
	tmpfile, err := ioutil.TempFile("", "skuid")
	if err != nil {
		return "", err
	}
	// write to our new file
	defer data.Close()
	if _, err := io.Copy(tmpfile, data); err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
}
