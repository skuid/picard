package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/cache"
	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/request"
	"github.com/skuid/warden/pkg/tokenMonkey"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var zlog = zap.L()

/*
LoadStreamed will load the request from SeaQuill, but will process it as a stream
of json objects rather than trying to process the whole thing at once.
*/
func LoadStreamed(ctx context.Context, w http.ResponseWriter, dataSource ds.DataSourceNew, reqBody map[string]interface{}, metadata interface{}) error {
	requestingMetadata := !reflect.ValueOf(metadata).IsNil()
	url := dataSource.AdapterAPIAddress() + "entity/load"
	headers := request.ProxyHeaders{
		SchemasOption: "true",
	}

	userInfo, err := api.UserInfoFromContext(ctx)

	if err != nil {
		zlog.Error(
			"There was an error obtaining information about the running user",
			zap.Error(err),
		)
		return err
	}

	redisConn := cache.GetConnection()
	defer redisConn.Close()
	connConfig, err := dataSource.ConnectionConfig(cache.New(redisConn), userInfo)

	if err != nil {
		zlog.Error(
			"There was an error obtaining data source access credentials",
			zap.Error(err),
		)
		return err
	}

	reqBody["database"] = connConfig["database"]

	req, err := request.BuildRequestWithBody(ctx, reqBody, "POST", url, headers)
	if err != nil {
		return err
	}

	if viper.GetBool("gzip_load_proxy") {
		req.Header.Add("Accept-Encoding", "gzip")
	}

	client := &http.Client{}

	cto := viper.GetDuration("client_timeout")
	if cto >= 0 {
		client.Timeout = cto
	}

	res, err := client.Do(req)

	if err != nil {
		zlog.Error(
			"There was an error in the streaming load response, getting data back from the proxy.",
			zap.Error(err),
		)
		return err
	}

	defer res.Body.Close()

	var r io.ReadCloser
	switch res.Header.Get("Content-Encoding") {
	case "gzip":
		if r, err = gzip.NewReader(res.Body); err != nil {
			return err
		}
		defer r.Close()
	default:
		r = res.Body
	}

	enc := json.NewEncoder(w)

	if _, err := w.Write([]byte("{")); err != nil {
		return err
	}
	if requestingMetadata {
		if _, err := w.Write([]byte("\"metadata\":")); err != nil {
			return err
		}
		if err := enc.Encode(metadata); err != nil {
			return err
		}
		if _, err := w.Write([]byte(",")); err != nil {
			return err
		}
	}
	if viper.GetBool("xtream") {
		b := new(bytes.Buffer)
		if _, err := b.ReadFrom(r); err != nil {
			return err
		}

		if _, err := b.ReadBytes('{'); err != nil {
			return err
		}

		// Write out the rest of the first chunk
		if _, err := b.WriteTo(w); err != nil {
			return err
		}

		// copy out the rest
		if _, err := io.Copy(w, res.Body); err != nil {
			return err
		}
	} else {
		dec := json.NewDecoder(r)
		tm := tokenMonkey.JSONer{
			Dec: dec,
			Enc: enc,
		}

		json := tm.TraverseObj("", func() interface{} {
			// just going to basically noop here. We shouldn't actually
			// get called since there shouldn't be any empty keys
			return nil
		})

		if tm.Err != nil {
			return tm.Err
		}
		if _, err := w.Write([]byte("\"models\":")); err != nil {
			return err
		}
		if err := enc.Encode(json.(map[string]interface{})["models"]); err != nil {
			return err
		}
		if _, err := w.Write([]byte("}")); err != nil {
			return err
		}
	}

	w.WriteHeader(res.StatusCode)
	return nil
}
