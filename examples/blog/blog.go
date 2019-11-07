package blog

import (
	"log"

	"github.com/skuid/picard"
	"github.com/skuid/picard/crypto"
	"github.com/skuid/picard/metadata"
	qp "github.com/skuid/picard/queryparts"
	"github.com/skuid/picard/tags"
)

// User example struct
type User struct {
	Metadata       metadata.Metadata `picard:"tablename=users"`
	ID             string            `picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `picard:"column=username"`

	Email    string `picard:"column=email"`
	Password string `picard:"encryptedcolumn=password"`
	Blogs    []Blog `picard:"child,foreign_key=UserID"`
}

// Blog example struct
type Blog struct {
	Metadata       metadata.Metadata `picard:"tablename=blogs"`
	ID             string            `picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `picard:"column=name"`
	Tags           []Tag             `picard:"child,foreign_key=BlogID"`
	UserID         string            `picard:"foreign_key,required,related=User,column=user_id"`
	User           User
}

// Tag example struct
type Tag struct {
	Metadata metadata.Metadata `picard:"tablename=tags"`
	ID       string            `picard:"primary_key,column=id"`
	Name     string            `picard:"column=name"`
	BlogID   string            `picard:"foreign_key,required,related=Blog,column=blog_id"`
	Blog     Blog
}

func insertBlog(p picard.ORM, newBlog *Blog) error {
	return p.CreateModel(newBlog)
}

func insertUser(p picard.ORM, newUser *User) error {
	return p.CreateModel(newUser)
}

func insertTag(p picard.ORM, newTag *Tag) error {
	return p.CreateModel(newTag)
}

// insert data creates users, blogs, and tags
func insertData(p picard.ORM) error {
	newUser := User{
		Name: "Deanna_Troi",
		ID:   "00000000-0000-0000-0000-000000000001",
	}

	if err := insertUser(p, &newUser); err != nil {
		return err
	}

	newBlogA := Blog{
		Name:   "Betazoid",
		ID:     "00000000-0000-0000-0000-000000000001",
		UserID: "00000000-0000-0000-0000-000000000001",
	}
	if err := insertBlog(p, &newBlogA); err != nil {
		return err
	}

	newBlogB := Blog{
		Name:   "Captain",
		ID:     "00000000-0000-0000-0000-000000000002",
		UserID: "00000000-0000-0000-0000-000000000001",
	}
	if err := insertBlog(p, &newBlogB); err != nil {
		return err
	}

	newTag := Tag{
		Name:   "space",
		ID:     "00000000-0000-0000-0000-000000000003",
		BlogID: "00000000-0000-0000-0000-000000000002",
	}
	return insertTag(p, &newTag)
}

// getAllBlogs grabs all blogs and eager logads Tag association in Tags field, ordering by Name in descending order
func getAllBlogs(p picard.ORM) ([]interface{}, error) {
	blogs, err := p.FilterModel(picard.FilterRequest{
		FilterModel: Blog{},
		Associations: []tags.Association{
			{
				Name: "Tag",
			},
		},
		OrderBy: []qp.OrderByRequest{
			{
				Field:      "Name",
				Descending: true,
			},
		},
	})

	if err != nil {
		return nil, err
	}
	return blogs, nil
}

/* getUser grabs a user(s) by name and eager loads association model fields.
 * Related Blog models are retrieved for Users in the Blogs field.
 * For Blog associations, the related Tag associations are fetched and added to the Tags field.
 * Only the ID and Name fields are retrieved for Users, Blogs, and Tags
 */
func getUser(p picard.ORM, name string) ([]interface{}, error) {
	filter := User{
		Name: name,
	}
	fields := []string{
		"ID",
		"Name",
	}
	users, err := p.FilterModel(picard.FilterRequest{
		FilterModel:  filter,
		SelectFields: fields,
		Associations: []tags.Association{
			{
				Name:         "Blog",
				SelectFields: fields,
				Associations: []tags.Association{
					{
						Name:         "Tag",
						SelectFields: fields,
					},
				},
			},
		},
	})

	if err != nil {
		return nil, err
	}
	return users, nil
}

func getBlog(p picard.ORM, name string) ([]interface{}, error) {
	blog, err := p.FilterModel(picard.FilterRequest{
		FilterModel: Blog{
			Name: name,
		},
		Associations: []tags.Association{
			{
				Name: "Tag",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return blog, nil
}

func getBlogs(p picard.ORM) ([]interface{}, error) {
	blogs, err := p.FilterModel(picard.FilterRequest{
		FilterModel: Blog{},
		FieldFilters: tags.OrFilterGroup{
			tags.FieldFilter{
				FieldName:   "Name",
				FilterValue: "Captain",
			},
			tags.FieldFilter{
				FieldName:   "UserID",
				FilterValue: "00000000-0000-0000-0000-000000000001",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return blogs, nil
}

func deleteBlog(p picard.ORM, name string) (int64, error) {
	rows, err := p.DeleteModel(Blog{
		Name: name,
	})
	if err != nil {
		return 0, err
	}
	return rows, nil
}

func updateBlog(p picard.ORM, id string, name string) error {
	err := p.SaveModel(Blog{
		ID:   id,
		Name: name,
	})
	if err != nil {
		return err
	}
	return nil
}

func main() {
	orgID := "00000000-0000-0000-0000-000000000001"
	userID := "00000000-0000-0000-0000-000000000001"
	crypto.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))
	picardORM := picard.New(orgID, userID)

	err := insertData(picardORM)
	if err != nil {
		log.Fatal(err)
	}

	blogs, err := getAllBlogs(picardORM)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("All blogs: %#v\n", blogs)

	user, err := getUser(picardORM, "Deanna_Troi")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("User: %#v\n", user)
	log.Printf("Nested child association (tags): %#v\n", user[0].(User).Blogs[0].Tags)

	blog, err := getBlog(picardORM, "Betazoid")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Blog: %#v\n", blog)
	log.Printf("Parent assocation (user): %#v\n", blog[0].(Blog).User)

	filterBlogs, err := getBlogs(picardORM)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Blogs for user 1 or with Captain name: %#v\n", filterBlogs)

	err = updateBlog(picardORM, "00000000-0000-0000-0000-000000000002", "Doctor")
	if err != nil {
		log.Fatal(err)
	}

	blogCount, err := deleteBlog(picardORM, "Captain")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("blogs deleted: %#v\n", blogCount)
}
