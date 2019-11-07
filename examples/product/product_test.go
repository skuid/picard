package product

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/skuid/picard/picard_test"
)

func TestSyncLuxInventory(t *testing.T) {
	testCases := []struct {
		desc            string
		mockProducts    []Product
		calledWith      []Product
		giveDeployError error
		wantError       error
	}{
		{
			"successfully deploy only lux products",
			[]Product{
				Product{
					Name:  "lux",
					Price: 20.00,
					Orders: []Order{
						Order{
							ID:       "25",
							Quantity: 9,
						},
					},
				},
				Product{
					Name:  "expensive",
					Price: 1000.00,
					Orders: []Order{
						Order{
							ID:       "22",
							Quantity: 1,
						},
					},
				},
				Product{
					Name:  "cheap",
					Price: 1.00,
					Orders: []Order{
						Order{
							ID:       "104",
							Quantity: 20,
						},
					},
				},
			},
			[]Product{
				Product{
					Name:  "lux",
					Price: 20.00,
					Orders: []Order{
						Order{
							ID:       "25",
							Quantity: 9,
						},
					},
				},
				Product{
					Name:  "expensive",
					Price: 1000.00,
					Orders: []Order{
						Order{
							ID:       "22",
							Quantity: 1,
						},
					},
				},
			},
			nil,
			nil,
		},
		{
			"fails to deploy products",
			[]Product{},
			[]Product{},
			errors.New("failed to insert user"),
			errors.New("failed to insert user"),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("SyncLuxInventory: %s", tc.desc), func(t *testing.T) {
			morm := &picard_test.MockORM{
				DeployError: tc.giveDeployError,
			}
			actualErr := syncLuxInventory(morm, tc.mockProducts)
			if !reflect.DeepEqual(tc.wantError, actualErr) {
				t.Errorf("Errors do not match.\n Actual: %#v\n expected %#v\n", actualErr, tc.wantError)
			}

			if !reflect.DeepEqual(morm.DeployCalledWith, tc.calledWith) {
				t.Errorf("Deploy not called with the right argument. Actual %#v\n expected %#v\n", morm.DeployCalledWith, tc.calledWith)
			}
		})
	}
}
