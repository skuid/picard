package main

import (
	"log"

	"github.com/skuid/picard"
)

// User example struct
type User struct {
	Metadata       picard.Metadata `picard:"tablename=user"`
	ID             string          `picard:"primary_key,column=id"`
	OrganizationID string          `picard:"multitenancy_key,column=organization_id"`
	Name           string          `picard:"lookup,column=name"`
	Posts          []Post          `picard:"child,foreign_key=UserID"`
}

// Post example struct
type Post struct {
	Metadata       picard.Metadata `picard:"tablename=message"`
	ID             string          `picard:"primary_key,column=id"`
	OrganizationID string          `picard:"multitenancy_key,column=organization_id"`
	Name           string          `picard:"lookup,column=name"`
	UserID         string          `picard:"foreign_key,lookup,required,related=User,column=user_id"`
	User           User            `validate:"-"`
	Tags           []Tag           `picard:"child,foreign_key=PostID"`
}

// Tag example struct
type Tag struct {
	Metadata picard.Metadata `picard:"tablename=tag"`
	ID       string          `picard:"primary_key,column=id"`
	Name     string          `picard:"lookup,column=name"`
	PostID   string          `picard:"foreign_key,lookup,required,related=Post,column=post_id"`
	Post     Post            `validate:"-"`
}

func doInserts(p picard.ORM) error {
	newUser := User{
		Name: "John Doe",
		ID:   "00000000-0000-0000-0000-000000000001",
	}

	if err := p.CreateModel(&newUser); err != nil {
		return err
	}

	newPost := Post{
		Name:   "foo bar",
		ID:     "00000000-0000-0000-0000-000000000002",
		UserID: "00000000-0000-0000-0000-000000000001",
	}

	if err := p.CreateModel(&newPost); err != nil {
		return err
	}

	newTag := Tag{
		Name:   "baz",
		ID:     "00000000-0000-0000-0000-000000000003",
		PostID: "00000000-0000-0000-0000-000000000002",
	}
	return p.CreateModel(&newTag)
}

func doLookup(p picard.ORM) ([]interface{}, error) {
	filter := User{
		Name: "JohnDoe",
	}
	results, err := p.FilterModelAssociations(filter, []string{"post.tag"})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func main() {
	orgID := "00000000-0000-0000-0000-000000000001"
	userID := "00000000-0000-0000-0000-000000000001"
	picardORM := picard.New(orgID, userID)
	doInserts(picardORM)
	results, err := doLookup(picardORM)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(results)
}
