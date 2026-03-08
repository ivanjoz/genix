package exec

import (
	"app/cloud"
	"fmt"
	"math/rand"
	"time"
)

// TestUsuario represents the entity to test the new Cloud ORM.
// It uses 'col' tags for mapping.
// 

type InnerStruct10 struct {
	Name string
	Value int32
	Valores2 []int32
}

type TestUsuario struct {
	ID        int32  `col:",sk"`         // Sort key for DynamoDB
	EmpresaID int32  `col:",pk"` // Partition key for DynamoDB / Primary key for D1
	Email     string `col:",index"`   // GSI Index
	Nombre    string `col:""`
	Valores1 []int32 `col:""`
	Valores2 []int32 `col:""`
	Complex1 InnerStruct10
	Complex2 []InnerStruct10
}

// RunCloudORMTest tests the generic ORM features with the 'cloudflare' provider.
func RunCloudORMTest() {
	fmt.Println("=== Starting Cloud ORM Test ===")

	// Test Init() function
	fmt.Println("\n--- Testing Init() ---")
	err := cloud.Init[TestUsuario]()
	if err != nil {
		fmt.Println("Init() Output:", err) // We expect this to output the "not implemented yet" along with queries (or an error)
	} else {
		fmt.Println("Init() Success")
	}

	// Create random data
	rand.Seed(time.Now().UnixNano())
	randomID := rand.Int31n(1000000)
	randomEmail := fmt.Sprintf("%d@example.com", randomID)

	newUser := TestUsuario{
		ID:        randomID,
		EmpresaID: 2,
		Email:     randomEmail,
		Nombre:    "Test Name",
	}

	// Test Insert()
	fmt.Println("\n--- Testing Insert() ---")
	err = cloud.Insert([]TestUsuario{newUser})
	if err != nil {
		fmt.Println("Insert() Output:", err)
	} else {
		fmt.Println("Insert() Success")
	}

	// Test Select by Email
	fmt.Println("\n--- Testing Select() by Email ---")
	var resultByEmail []TestUsuario
	err = cloud.Select(&resultByEmail).Where("email").Equals(randomEmail).Exec()
	if err != nil {
		fmt.Println("Select() by Email Output:", err)
	} else {
		fmt.Printf("Select() found %d items\n", len(resultByEmail))
	}

	// Test GetByID
	fmt.Println("\n--- Testing GetByID() ---")
	// The GetByID function expects an instance of the struct populated with the primary keys.
	searchUser := TestUsuario{
		EmpresaID: 2,
		ID:        randomID,
	}

	foundUser, err := cloud.GetByID(searchUser)
	if err != nil {
		fmt.Println("GetByID() Output:", err)
	} else {
		fmt.Printf("GetByID() Success: %+v\n", foundUser)
	}

	fmt.Println("\n=== Cloud ORM Test Finished ===")
}
