package product

import (
	"log"
	"time"

	"github.com/skuid/picard"
	"github.com/skuid/picard/crypto"
	"github.com/skuid/picard/metadata"
	"github.com/skuid/picard/tags"

	qp "github.com/skuid/picard/queryparts"
)

type Product struct {
	Metadata metadata.Metadata `picard:"tablename=products"`
	ID       string            `picard:"primary_key,column=id"`
	StoreID  string            `picard:"multitenancy_key,column=store_id"`
	Name     string            `picard:"column=username"`
	Price    float64           `picard:"column=price"`
	Orders   []Order           `picard:"child,foreign_key=ProductID"`

	CreatedByID string    `picard:"column=created_by_id,audit=created_by"`
	UpdatedByID string    `picard:"column=updated_by_id,audit=updated_by"`
	CreatedDate time.Time `picard:"column=created_at,audit=created_at"`
	UpdatedDate time.Time `picard:"column=updated_at,audit=updated_at"`
}

type Order struct {
	Metadata    metadata.Metadata `picard:"tablename=orders"`
	ID          string            `picard:"primary_key,column=id"`
	StoreID     string            `picard:"multitenancy_key,column=store_id"`
	ProductID   string            `picard:"foreign_key,required,related=Product,column=product_id"`
	Product     Product
	Quantity    int       `picard:"column=quantity"`
	CustomerID  string    `picard:"column=customer_id"`
	CreatedDate time.Time `picard:"column=created_at,audit=created_at"`
	UpdatedDate time.Time `picard:"column=updated_at,audit=updated_at"`
}

func syncLuxInventory(porm picard.ORM, products []Product) error {
	luxProducts := make([]Product, 0, len(products))
	for _, p := range products {
		if p.Price > 15.00 {
			luxProducts = append(luxProducts, p)
		}
	}
	return porm.Deploy(luxProducts)
}

func main() {
	storeID := "00000000-0000-0000-0000-000000000001"
	userID := "00000000-0000-0000-0000-000000000001"
	crypto.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))
	picardORM := picard.New(storeID, userID)

	inevntory := []Product{
		Product{
			Name:  "new tea",
			Price: 12.15,
			Orders: []Order{
				Order{
					Quantity:   1,
					CustomerID: "00000000-0000-0000-0000-000000000001",
				},
				Order{
					Quantity:   6,
					CustomerID: "00000000-0000-0000-0000-000000000002",
				},
				Order{
					Quantity:   30,
					CustomerID: "00000000-0000-0000-0000-000000000003",
				},
			},
		},
		Product{
			Name:  "no orders soap",
			Price: 5.99,
		},
		Product{
			Name:  "existing potted plant",
			Price: 19.95,
			Orders: []Order{
				Order{
					ID: "00000000-0000-0000-0000-000000000022",
				},
				Order{
					ID: "00000000-0000-0000-0000-000000000049",
				},
			},
		},
	}
	err := syncLuxInventory(picardORM, inevntory)
	if err != nil {
		log.Fatal(err)
	}
	products, err := picardORM.FilterModel(picard.FilterRequest{
		FilterModel: Product{},
		SelectFields: []string{
			"ID",
			"Name",
			"Price",
		},
		Associations: []tags.Association{
			{
				Name: "Order",
				SelectFields: []string{
					"ID",
					"CustomerID",
				},
			},
		},
		OrderBy: []qp.OrderByRequest{
			{
				Field:      "Price",
				Descending: true,
			},
		},
	})
	log.Printf("All deployed products: %#v\n", products)
}
