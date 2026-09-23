package sites

import "testing"

func TestCreateSiteValidation(t *testing.T) {
	req := CreateSiteRequest{CustomerID: 1, Name: "Sukhumvit Branch", Address: "Sukhumvit 38"}
	if err := req.Validate(); err != nil {
		t.Fatalf("valid site rejected: %v", err)
	}
	if req.Status != "active" {
		t.Fatalf("default status should be active, got %q", req.Status)
	}
}

func TestCreateSiteRequiresCustomerAndAddress(t *testing.T) {
	req := CreateSiteRequest{Name: "X", Address: "Y"}
	if err := req.Validate(); err == nil {
		t.Fatal("site without customer must be rejected")
	}
	req = CreateSiteRequest{CustomerID: 1, Name: "X"}
	if err := req.Validate(); err == nil {
		t.Fatal("site without address must be rejected")
	}
}

func TestCreateSiteLatLngBounds(t *testing.T) {
	bad := 200.0
	req := CreateSiteRequest{CustomerID: 1, Name: "X", Address: "Y", Latitude: &bad}
	if err := req.Validate(); err == nil {
		t.Fatal("latitude out of range must be rejected")
	}
	lng := 200.0
	req = CreateSiteRequest{CustomerID: 1, Name: "X", Address: "Y", Longitude: &lng}
	if err := req.Validate(); err == nil {
		t.Fatal("longitude out of range must be rejected")
	}
}
