//go:build !js

package webpage

import (
	"app/core"
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

const testDistribution = "d111111abcdef8.cloudfront.net"

// fakeRoute53 guarda los registros como lo haría la zona: CREATE falla si el nombre y tipo ya
// existen, y List devuelve los nombres con el punto final.
type fakeRoute53 struct {
	records []route53Types.ResourceRecordSet
	changes []route53Types.Change
}

func (fake *fakeRoute53) ChangeResourceRecordSets(_ context.Context, input *route53.ChangeResourceRecordSetsInput, _ ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
	for _, change := range input.ChangeBatch.Changes {
		if change.Action == route53Types.ChangeActionCreate && fake.find(change.ResourceRecordSet) >= 0 {
			return nil, &route53Types.InvalidChangeBatch{Message: aws.String("already exists")}
		}
	}
	for _, change := range input.ChangeBatch.Changes {
		fake.changes = append(fake.changes, change)
		record := *change.ResourceRecordSet
		if change.Action == route53Types.ChangeActionCreate {
			record.Name = aws.String(aws.ToString(record.Name) + ".")
			fake.records = append(fake.records, record)
		} else if index := fake.find(&record); index >= 0 {
			fake.records = append(fake.records[:index], fake.records[index+1:]...)
		}
	}
	return &route53.ChangeResourceRecordSetsOutput{}, nil
}

func (fake *fakeRoute53) ListResourceRecordSets(_ context.Context, input *route53.ListResourceRecordSetsInput, _ ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
	start := normalizeDNSName(aws.ToString(input.StartRecordName))
	output := []route53Types.ResourceRecordSet{}
	for _, record := range fake.records {
		// Suficiente para los tests: el orden real de Route 53 pone los demás nombres después.
		if normalizeDNSName(aws.ToString(record.Name)) >= start {
			output = append(output, record)
		}
	}
	return &route53.ListResourceRecordSetsOutput{ResourceRecordSets: output}, nil
}

func (fake *fakeRoute53) find(target *route53Types.ResourceRecordSet) int {
	for index, record := range fake.records {
		if normalizeDNSName(aws.ToString(record.Name)) == normalizeDNSName(aws.ToString(target.Name)) && record.Type == target.Type {
			return index
		}
	}
	return -1
}

func setupFakeRoute53(t *testing.T) *fakeRoute53 {
	t.Helper()
	core.Env = &core.EnvStruct{
		STOREFRONT_HOSTED_ZONE_ID:      "/hostedzone/ZTEST",
		STOREFRONT_DISTRIBUTION_DOMAIN: testDistribution,
	}
	fake := &fakeRoute53{}
	previousFactory := newRoute53Client
	newRoute53Client = func() route53API { return fake }
	t.Cleanup(func() { newRoute53Client = previousFactory })
	return fake
}

func TestProvisionStorefrontDomainCreatesAliases(t *testing.T) {
	fake := setupFakeRoute53(t)

	if err := provisionStorefrontDomain("tienda.un.pe", false); err != nil {
		t.Fatalf("provision: %v", err)
	}
	if len(fake.records) != 2 {
		t.Fatalf("se esperaban A y AAAA, hay %d registros", len(fake.records))
	}
	for _, record := range fake.records {
		if record.AliasTarget == nil || aws.ToString(record.AliasTarget.DNSName) != testDistribution ||
			aws.ToString(record.AliasTarget.HostedZoneId) != cloudFrontHostedZoneID {
			t.Errorf("alias incorrecto: %+v", record.AliasTarget)
		}
	}
}

func TestProvisionStorefrontDomainRejectsTakenName(t *testing.T) {
	fake := setupFakeRoute53(t)
	fake.records = append(fake.records, route53Types.ResourceRecordSet{
		Name: aws.String("tienda.un.pe."), Type: route53Types.RRTypeA,
		ResourceRecords: []route53Types.ResourceRecord{{Value: aws.String("1.2.3.4")}},
	})

	// Ni siquiera con allowExisting: el nombre es de otro servicio de la zona.
	err := provisionStorefrontDomain("tienda.un.pe", true)
	if err == nil || !strings.Contains(err.Error(), "ya está registrado") {
		t.Fatalf("se esperaba 'ya está registrado', llegó %v", err)
	}
}

func TestProvisionStorefrontDomainAllowsOwnAliasOnlyWhenAsked(t *testing.T) {
	setupFakeRoute53(t)
	if err := provisionStorefrontDomain("tienda.un.pe", false); err != nil {
		t.Fatalf("primer alta: %v", err)
	}

	// Re-guardar el mismo dominio reintenta la publicación.
	if err := provisionStorefrontDomain("tienda.un.pe", true); err != nil {
		t.Fatalf("re-guardado: %v", err)
	}
	// Otra company pidiendo el mismo nombre.
	if err := provisionStorefrontDomain("tienda.un.pe", false); err == nil {
		t.Fatal("un dominio de otra tienda no debe aceptarse")
	}
}

func TestRemoveStorefrontDomainOnlyDeletesOwnAliases(t *testing.T) {
	fake := setupFakeRoute53(t)
	if err := provisionStorefrontDomain("tienda.un.pe", false); err != nil {
		t.Fatalf("alta: %v", err)
	}
	fake.records = append(fake.records, route53Types.ResourceRecordSet{
		Name: aws.String("tienda.un.pe."), Type: route53Types.RRTypeTxt,
		ResourceRecords: []route53Types.ResourceRecord{{Value: aws.String(`"verificacion"`)}},
	})

	if err := removeStorefrontDomain("tienda.un.pe"); err != nil {
		t.Fatalf("baja: %v", err)
	}
	if len(fake.records) != 1 || fake.records[0].Type != route53Types.RRTypeTxt {
		t.Fatalf("solo debía quedar el TXT ajeno, quedan %+v", fake.records)
	}
	// Reintentar sin nada que borrar no falla ni envía cambios.
	changesBefore := len(fake.changes)
	if err := removeStorefrontDomain("tienda.un.pe"); err != nil {
		t.Fatalf("reintento: %v", err)
	}
	if len(fake.changes) != changesBefore {
		t.Fatal("el reintento no debía enviar cambios")
	}
}

func TestStorefrontDNSRequiresConfig(t *testing.T) {
	core.Env = &core.EnvStruct{}
	if err := provisionStorefrontDomain("tienda.un.pe", false); err == nil {
		t.Fatal("sin configuración debía fallar")
	}
}
