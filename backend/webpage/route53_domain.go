//go:build !js

package webpage

import (
	"app/core"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// Cada tienda es un par de alias A/AAAA en la zona de Route 53 que apuntan a la distribución de
// CloudFront de las tiendas (cloud/storefront.yml). El certificado es un comodín *.ZONE_NAME, así
// que dar de alta un hostname no emite nada: basta con el registro DNS.
//
// La reserva del nombre la hace Route 53 y no la base de datos: CREATE es atómico y falla con
// InvalidChangeBatch si el nombre ya tiene un registro (de otra tienda o de cualquier otra cosa
// de la zona, como un CNAME). Es el mismo contrato que tenían los Worker Custom Domains.

// cloudFrontHostedZoneID es fijo: es la zona que AWS usa para todo alias hacia CloudFront.
const cloudFrontHostedZoneID = "Z2FDTNDATAQYW2"

var storefrontAliasTypes = []route53Types.RRType{route53Types.RRTypeA, route53Types.RRTypeAaaa}

// route53API es lo único que este archivo usa del cliente, para poder sustituirlo en los tests.
type route53API interface {
	ChangeResourceRecordSets(context.Context, *route53.ChangeResourceRecordSetsInput, ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error)
	ListResourceRecordSets(context.Context, *route53.ListResourceRecordSetsInput, ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error)
}

var newRoute53Client = func() route53API {
	return route53.NewFromConfig(core.GetAwsConfig())
}

type storefrontDNSConfig struct {
	hostedZoneID       string
	distributionDomain string
}

func loadStorefrontDNSConfig() (storefrontDNSConfig, error) {
	dnsConfig := storefrontDNSConfig{
		hostedZoneID:       strings.TrimPrefix(strings.TrimSpace(core.Env.STOREFRONT_HOSTED_ZONE_ID), "/hostedzone/"),
		distributionDomain: normalizeDNSName(core.Env.STOREFRONT_DISTRIBUTION_DOMAIN),
	}
	if dnsConfig.hostedZoneID == "" || dnsConfig.distributionDomain == "" {
		return dnsConfig, errors.New("storefront.hosted_zone_id y storefront.distribution_domain son requeridos")
	}
	return dnsConfig, nil
}

// provisionStorefrontDomain rechaza los nombres ocupados y crea el alias de los disponibles.
// allowExistingStorefrontDomain acepta un nombre que ya apunta a la distribución de las tiendas:
// es el re-guardado del mismo dominio, que sirve para reintentar una publicación.
func provisionStorefrontDomain(hostname string, allowExistingStorefrontDomain bool) error {
	dnsConfig, configError := loadStorefrontDNSConfig()
	if configError != nil {
		return configError
	}
	client := newRoute53Client()
	requestContext, cancelRequest := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelRequest()

	changes := make([]route53Types.Change, 0, len(storefrontAliasTypes))
	for _, recordType := range storefrontAliasTypes {
		changes = append(changes, route53Types.Change{
			Action:            route53Types.ChangeActionCreate,
			ResourceRecordSet: storefrontAliasRecord(hostname, recordType, dnsConfig.distributionDomain),
		})
	}
	_, changeError := client.ChangeResourceRecordSets(requestContext, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(dnsConfig.hostedZoneID),
		ChangeBatch: &route53Types.ChangeBatch{
			Comment: aws.String("Tienda genix " + hostname),
			Changes: changes,
		},
	})
	if changeError == nil {
		core.Log("Dominio Route 53 registrado::", hostname)
		return nil
	}

	var invalidBatch *route53Types.InvalidChangeBatch
	if !errors.As(changeError, &invalidBatch) {
		return fmt.Errorf("error creando el registro DNS: %w", changeError)
	}

	// El nombre está ocupado. Solo se acepta si es nuestro y el llamador lo permite.
	existingRecords, listError := findHostnameRecords(requestContext, client, dnsConfig.hostedZoneID, hostname)
	if listError != nil {
		return listError
	}
	if allowExistingStorefrontDomain && isStorefrontAlias(existingRecords, dnsConfig.distributionDomain) {
		core.Log("Dominio Route 53 ya registrado::", hostname)
		return nil
	}
	if len(existingRecords) > 0 {
		return fmt.Errorf("el dominio %s ya está registrado", hostname)
	}
	return fmt.Errorf("Route 53 rechazó el dominio %s: %w", hostname, changeError)
}

// removeStorefrontDomain libera un hostname que la company dejó de usar. Solo borra los alias que
// apuntan a la distribución de las tiendas: cualquier otro registro con ese nombre no es de esta
// tienda. Sin nada que borrar no hace nada, y eso es lo que permite reintentarla.
func removeStorefrontDomain(hostname string) error {
	dnsConfig, configError := loadStorefrontDNSConfig()
	if configError != nil {
		return configError
	}
	client := newRoute53Client()
	requestContext, cancelRequest := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelRequest()

	existingRecords, listError := findHostnameRecords(requestContext, client, dnsConfig.hostedZoneID, hostname)
	if listError != nil {
		return listError
	}

	changes := []route53Types.Change{}
	for index := range existingRecords {
		if isStorefrontAliasRecord(existingRecords[index], dnsConfig.distributionDomain) {
			// Se borra el registro tal como lo devolvió Route 53: un DELETE tiene que coincidir
			// exactamente con lo guardado.
			changes = append(changes, route53Types.Change{
				Action:            route53Types.ChangeActionDelete,
				ResourceRecordSet: &existingRecords[index],
			})
		}
	}
	if len(changes) == 0 {
		return nil
	}

	_, changeError := client.ChangeResourceRecordSets(requestContext, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(dnsConfig.hostedZoneID),
		ChangeBatch:  &route53Types.ChangeBatch{Comment: aws.String("Baja tienda genix " + hostname), Changes: changes},
	})
	if changeError != nil {
		return fmt.Errorf("error borrando el registro DNS de %s: %w", hostname, changeError)
	}
	core.Log("Registro DNS borrado::", hostname, len(changes))
	return nil
}

func storefrontAliasRecord(hostname string, recordType route53Types.RRType, distributionDomain string) *route53Types.ResourceRecordSet {
	return &route53Types.ResourceRecordSet{
		Name: aws.String(hostname),
		Type: recordType,
		AliasTarget: &route53Types.AliasTarget{
			DNSName:              aws.String(distributionDomain),
			HostedZoneId:         aws.String(cloudFrontHostedZoneID),
			EvaluateTargetHealth: false,
		},
	}
}

// findHostnameRecords devuelve todos los registros con exactamente ese nombre. ListResourceRecordSets
// empieza en StartRecordName y sigue en orden, así que se corta en cuanto el nombre cambia.
func findHostnameRecords(requestContext context.Context, client route53API, hostedZoneID string, hostname string) ([]route53Types.ResourceRecordSet, error) {
	response, listError := client.ListResourceRecordSets(requestContext, &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(hostedZoneID),
		StartRecordName: aws.String(hostname),
		MaxItems:        aws.Int32(20),
	})
	if listError != nil {
		return nil, fmt.Errorf("error consultando los registros DNS de %s: %w", hostname, listError)
	}
	records := []route53Types.ResourceRecordSet{}
	for _, record := range response.ResourceRecordSets {
		if normalizeDNSName(aws.ToString(record.Name)) != normalizeDNSName(hostname) {
			break
		}
		records = append(records, record)
	}
	return records, nil
}

// isStorefrontAlias exige que haya al menos un registro y que todos sean alias de las tiendas: un
// nombre con un alias nuestro y además un TXT o MX ajeno no se toma como propio.
func isStorefrontAlias(records []route53Types.ResourceRecordSet, distributionDomain string) bool {
	if len(records) == 0 {
		return false
	}
	for _, record := range records {
		if !isStorefrontAliasRecord(record, distributionDomain) {
			return false
		}
	}
	return true
}

func isStorefrontAliasRecord(record route53Types.ResourceRecordSet, distributionDomain string) bool {
	if record.Type != route53Types.RRTypeA && record.Type != route53Types.RRTypeAaaa {
		return false
	}
	return record.AliasTarget != nil &&
		normalizeDNSName(aws.ToString(record.AliasTarget.DNSName)) == distributionDomain
}

// Route 53 devuelve los nombres con el punto final y CloudFront sin él.
func normalizeDNSName(name string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
}
