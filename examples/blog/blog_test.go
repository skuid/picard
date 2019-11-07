package blog

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/skuid/picard"
	"github.com/skuid/picard/picard_test"
	qp "github.com/skuid/picard/queryparts"
	"github.com/skuid/picard/tags"
)

func TestInsertUser(t *testing.T) {
	testCases := []struct {
		desc            string
		mockUser        User
		giveCreateError error
		wantError       error
	}{
		{
			"successfully inserts user",
			User{
				Name: "happy",
			},
			nil,
			nil,
		},
		{
			"fails to insert user",
			User{
				Name: "sad",
			},
			errors.New("failed to insert user"),
			errors.New("failed to insert user"),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("InsertUser: %s", tc.desc), func(t *testing.T) {
			morm := &picard_test.MockORM{
				CreateModelError: tc.giveCreateError,
			}
			actualErr := insertUser(morm, &tc.mockUser)
			if !reflect.DeepEqual(tc.wantError, actualErr) {
				t.Errorf("Errors do not match. Actual %#v expected %#v not equal", actualErr, tc.wantError)
			}

			if !reflect.DeepEqual(morm.CreateModelCalledWith, &tc.mockUser) {
				t.Errorf("Create model not called with the right argument. Actual %#v expected %#v not equal", morm.CreateModelCalledWith, tc.mockUser)
			}
		})
	}
}

func TestInsertBlog(t *testing.T) {
	testCases := []struct {
		desc            string
		mockBlog        Blog
		giveCreateError error
		wantError       error
	}{
		{
			"successfully inserts blog",
			Blog{
				Name: "happy",
			},
			nil,
			nil,
		},
		{
			"fails to insert blog",
			Blog{
				Name: "sad",
			},
			errors.New("failed to insert blog"),
			errors.New("failed to insert blog"),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("InsertBlog: %s", tc.desc), func(t *testing.T) {
			morm := &picard_test.MockORM{
				CreateModelError: tc.giveCreateError,
			}
			actualErr := insertBlog(morm, &tc.mockBlog)
			if !reflect.DeepEqual(tc.wantError, actualErr) {
				t.Errorf("Errors do not match. Actual %#v expected %#v not equal", actualErr, tc.wantError)
			}

			if !reflect.DeepEqual(morm.CreateModelCalledWith, &tc.mockBlog) {
				t.Errorf("Create model not called with the right argument. Actual %#v expected %#v not equal", morm.CreateModelCalledWith, tc.mockBlog)
			}
		})
	}
}

func TestInsertTag(t *testing.T) {
	testCases := []struct {
		desc            string
		mockTag         Tag
		giveCreateError error
		wantError       error
	}{
		{
			"successfully inserts tag",
			Tag{
				Name: "happy",
			},
			nil,
			nil,
		},
		{
			"fails to insert tag",
			Tag{
				Name: "sad",
			},
			errors.New("failed to insert user"),
			errors.New("failed to insert user"),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("InsertTag: %s", tc.desc), func(t *testing.T) {
			morm := &picard_test.MockORM{
				CreateModelError: tc.giveCreateError,
			}
			actualErr := insertTag(morm, &tc.mockTag)
			if !reflect.DeepEqual(tc.wantError, actualErr) {
				t.Errorf("Errors do not match. Actual %#v expected %#v not equal", actualErr, tc.wantError)
			}

			if !reflect.DeepEqual(morm.CreateModelCalledWith, &tc.mockTag) {
				t.Errorf("Create model not called with the right argument. Actual %#v expected %#v not equal", morm.CreateModelCalledWith, tc.mockTag)
			}
		})
	}
}

func TestGetAllBlogs(t *testing.T) {

	testCases := []struct {
		desc              string
		giveFilterReturns []interface{}
		giveFilterError   error
		wantResults       []interface{}
		wantError         error
	}{
		{
			"retrieve all the blogs",
			[]interface{}{
				Blog{
					Name:   "Betazoid",
					ID:     "00000000-0000-0000-0000-000000000001",
					UserID: "00000000-0000-0000-0000-000000000001",
				},
			},
			nil,
			[]interface{}{
				Blog{
					Name:   "Betazoid",
					ID:     "00000000-0000-0000-0000-000000000001",
					UserID: "00000000-0000-0000-0000-000000000001",
				},
			},
			nil,
		},
		{
			"fails to retrieve blogs",
			[]interface{}(nil),
			errors.New("Filter error"),
			[]interface{}(nil),
			errors.New("Filter error"),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("GetAllBlogs: %s", tc.desc), func(t *testing.T) {
			morm := &picard_test.MockORM{
				FilterModelReturns: tc.giveFilterReturns,
				FilterModelError:   tc.giveFilterError,
			}
			actualResults, actualErr := getAllBlogs(morm)
			if !reflect.DeepEqual(tc.wantResults, actualResults) {
				t.Errorf("Results do not match. Actual: %#v\nExpected: %#v\n", actualResults, tc.wantResults)
			}
			if !reflect.DeepEqual(tc.wantError, actualErr) {
				t.Errorf("Errors do not match.\n Actual: %#v\nExpected %#v\n", actualErr, tc.wantError)
			}

			wantCallWith := picard.FilterRequest{
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
			}
			if !reflect.DeepEqual(morm.FilterModelCalledWith, wantCallWith) {
				t.Errorf("Filter request not called with the right argument. Actual: %#v\nExpected %#v\n", morm.FilterModelCalledWith, wantCallWith)
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	testCases := []struct {
		desc            string
		giveName        string
		wantReturns     bool
		giveFilterError error
		wantError       error
	}{
		{
			"successfully retrieve a user",
			"robo",
			true,
			nil,
			nil,
		},
		{
			"fails to retrieve user",
			"sad",
			false,
			errors.New("Filter error"),
			errors.New("Filter error"),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("getUser: %s", tc.desc), func(t *testing.T) {
			mockUser := User{
				Name: tc.giveName,
				ID:   "00000000-0000-0000-0000-000000000001",
				Blogs: []Blog{
					{
						Name: "magic ways",
						ID:   "00000000-0000-0000-0000-000000000001",
						Tags: []Tag{
							Tag{
								Name: "silver",
								ID:   "00000000-0000-0000-0000-000000000001",
							},
						},
					},
				},
			}

			var filterReturns []interface{}
			if tc.wantReturns {
				filterReturns = []interface{}{
					mockUser,
				}
			}

			morm := &picard_test.MockORM{
				FilterModelReturns: filterReturns,
				FilterModelError:   tc.giveFilterError,
			}
			actualResults, actualErr := getUser(morm, tc.giveName)
			if tc.wantReturns {
				wantResults := []interface{}{
					mockUser,
				}
				if !reflect.DeepEqual(wantResults, actualResults) {
					t.Errorf("Results do not match. Actual: %#v\nExpected: %#v\n", actualResults, wantResults)
				}
			}
			if !reflect.DeepEqual(tc.wantError, actualErr) {
				t.Errorf("Errors do not match.\n Actual: %#v\nExpected %#v\n", actualErr, tc.wantError)
			}

			expectCalledWith := picard.FilterRequest{
				FilterModel: User{
					Name: mockUser.Name,
				},
				SelectFields: []string{
					"ID",
					"Name",
				},
				Associations: []tags.Association{
					{
						Name: "Blog",
						SelectFields: []string{
							"ID",
							"Name",
						},
						Associations: []tags.Association{
							{
								Name: "Tag",
								SelectFields: []string{
									"ID",
									"Name",
								},
							},
						},
					},
				},
			}

			if !reflect.DeepEqual(morm.FilterModelCalledWith, expectCalledWith) {
				t.Errorf("Filter request not called with the right argument.\n Actual: %#v\n Expected: %#v \n", morm.FilterModelCalledWith, expectCalledWith)
			}
		})
	}
}

func TestGetBlog(t *testing.T) {
	testCases := []struct {
		desc            string
		giveName        string
		wantReturns     bool
		giveFilterError error
		wantError       error
	}{
		{
			"successfully retrieve a blog",
			"happy",
			true,
			nil,
			nil,
		},
		{
			"fails to retrieve blog",
			"sad",
			false,
			errors.New("Filter error"),
			errors.New("Filter error"),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("getBlog: %s", tc.desc), func(t *testing.T) {
			mockBlog := Blog{
				Name: tc.giveName,
				ID:   "00000000-0000-0000-0000-000000000001",
				Tags: []Tag{
					Tag{
						Name: "silver",
						ID:   "00000000-0000-0000-0000-000000000001",
					},
				},
			}

			var filterReturns []interface{}
			if tc.wantReturns {
				filterReturns = []interface{}{
					mockBlog,
				}
			}

			morm := &picard_test.MockORM{
				FilterModelReturns: filterReturns,
				FilterModelError:   tc.giveFilterError,
			}
			actualResults, actualErr := getBlog(morm, tc.giveName)
			if tc.wantReturns {
				wantResults := []interface{}{
					mockBlog,
				}
				if !reflect.DeepEqual(wantResults, actualResults) {
					t.Errorf("Results do not match. Actual: %#v\nExpected: %#v\n", actualResults, wantResults)
				}
			}
			if !reflect.DeepEqual(tc.wantError, actualErr) {
				t.Errorf("Errors do not match.\n Actual: %#v\nExpected %#v\n", actualErr, tc.wantError)
			}

			expectCalledWith := picard.FilterRequest{
				FilterModel: Blog{
					Name: mockBlog.Name,
				},
				Associations: []tags.Association{
					{
						Name: "Tag",
					},
				},
			}

			if !reflect.DeepEqual(morm.FilterModelCalledWith, expectCalledWith) {
				t.Errorf("Filter request not called with the right argument.\n Actual: %#v\n Expected: %#v \n", morm.FilterModelCalledWith, expectCalledWith)
			}
		})
	}
}

func TestUpdateBlog(t *testing.T) {
	testCases := []struct {
		desc            string
		giveID          string
		giveName        string
		wantReturns     bool
		giveUpdateError error
		wantError       error
	}{
		{
			"successfully update a blog",
			"00000000-0000-0000-0000-000000000001",
			"happy",
			true,
			nil,
			nil,
		},
		{
			"fails to update blog",
			"00000000-0000-0000-0000-000000000000",
			"sad",
			false,
			errors.New("update error"),
			errors.New("update error"),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("updateBlog: %s", tc.desc), func(t *testing.T) {
			mockBlog := Blog{
				Name: tc.giveName,
				ID:   tc.giveID,
				Tags: []Tag{
					Tag{
						Name: "silver",
						ID:   "00000000-0000-0000-0000-000000000001",
					},
				},
			}

			morm := &picard_test.MockORM{
				SaveModelError: tc.giveUpdateError,
			}
			actualErr := updateBlog(morm, tc.giveID, tc.giveName)
			if !reflect.DeepEqual(tc.wantError, actualErr) {
				t.Errorf("Errors do not match.\n Actual: %#v\nExpected %#v\n", actualErr, tc.wantError)
			}

			expectCalledWith := Blog{
				Name: mockBlog.Name,
				ID:   mockBlog.ID,
			}

			if !reflect.DeepEqual(morm.SaveModelCalledWith, expectCalledWith) {
				t.Errorf("SaveModel not called with the right argument.\n Actual: %#v\n Expected: %#v \n", morm.SaveModelCalledWith, expectCalledWith)
			}
		})
	}
}

func TestDeleteBlog(t *testing.T) {
	testCases := []struct {
		desc            string
		giveName        string
		wantReturns     bool
		giveDeleteError error
		wantDeleteCount int64
		wantError       error
	}{
		{
			"successfully delete a blog",
			"happy",
			true,
			nil,
			1,
			nil,
		},
		{
			"fails to delete blog",
			"sad",
			false,
			errors.New("Filter error"),
			0,
			errors.New("Filter error"),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("deleteBlog: %s", tc.desc), func(t *testing.T) {
			mockBlog := Blog{
				Name: tc.giveName,
				Tags: []Tag{
					Tag{
						Name: "silver",
						ID:   "00000000-0000-0000-0000-000000000001",
					},
				},
			}

			morm := &picard_test.MockORM{
				DeleteModelError:        tc.giveDeleteError,
				DeleteModelRowsAffected: tc.wantDeleteCount,
			}
			count, actualErr := deleteBlog(morm, tc.giveName)
			if tc.wantDeleteCount != count {
				t.Errorf("Count is not equal.\nExpected %#v Got %#v", tc.wantDeleteCount, count)
			}
			if !reflect.DeepEqual(tc.wantError, actualErr) {
				t.Errorf("Errors do not match.\n Actual: %#v\nExpected %#v\n", actualErr, tc.wantError)
			}

			expectCalledWith := Blog{
				Name: mockBlog.Name,
				ID:   mockBlog.ID,
			}

			if !reflect.DeepEqual(morm.DeleteModelCalledWith, expectCalledWith) {
				t.Errorf("DeleteModel not called with the right argument.\n Actual: %#v\n Expected: %#v \n", morm.DeleteModelCalledWith, expectCalledWith)
			}
		})
	}
}
