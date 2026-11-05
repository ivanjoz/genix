package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfnTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// Infraestructura de las tiendas (<tienda>.frontend.zone_name): certificado comodín, bucket del
// HTML, CloudFront con su función de ruteo y los permisos del rol de las Lambdas. Ver
// storefront.yml. Es un stack aparte del principal para poder desplegarlo sin tocar las Lambdas.
//
//go:embed storefront.yml
var storefrontTemplate string

// CloudFront solo acepta certificados de ACM emitidos en us-east-1, y el certificado vive en el
// mismo stack que la distribución.
const storefrontRegion = "us-east-1"

func DeployStorefront(params DeployParams) {
	zoneName := strings.TrimSpace(params.Frontend.ZoneName)
	hostedZoneID := strings.TrimPrefix(strings.TrimSpace(params.Storefront.HostedZoneID), "/hostedzone/")
	roleARN := strings.TrimSpace(params.AWS.LambdaIAMRole)
	if zoneName == "" || hostedZoneID == "" || roleARN == "" {
		exitWithDeployError("frontend.zone_name, storefront.hosted_zone_id y aws.lambda_iam_role son requeridos para la acción 5.")
	}
	// El ARN del rol es arn:aws:iam::<cuenta>:role/<ruta>/<nombre>; RolePolicy pide el nombre.
	roleName := roleARN[strings.LastIndex(roleARN, "/")+1:]

	stackName := params.AppName + "-storefront"
	awsConfig, err := MakeAwsConfig(params.AWS.Profile, storefrontRegion)
	if err != nil {
		panic("Error al cargar la configuración de AWS: " + err.Error())
	}
	client := cloudformation.NewFromConfig(awsConfig)
	ctx := context.TODO()

	templateParameters := []cfnTypes.Parameter{
		makeStackParameter("NamePrefix", params.AppName),
		makeStackParameter("ZoneName", zoneName),
		makeStackParameter("HostedZoneId", hostedZoneID),
		makeStackParameter("LambdaRoleName", roleName),
	}
	capabilities := []cfnTypes.Capability{cfnTypes.CapabilityCapabilityIam}

	currentStatus, stackExists := DescribeStackStatus(ctx, client, stackName)
	if stackExists && currentStatus == cfnTypes.StackStatusRollbackComplete {
		exitWithDeployError(fmt.Sprintf("El stack %v está en ROLLBACK_COMPLETE. Bórralo antes de reintentar.", stackName))
	}

	deployStartedAt := time.Now().Add(-stackEventsPollInterval)
	if stackExists {
		fmt.Printf("Actualizando stack %v (estado actual: %v)...\n", stackName, currentStatus)
		_, err = client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
			StackName: &stackName, TemplateBody: &storefrontTemplate,
			Parameters: templateParameters, Capabilities: capabilities,
		})
		if err != nil && strings.Contains(err.Error(), "No updates are to be performed") {
			fmt.Println("Sin cambios en la infraestructura de tiendas.")
			syncStorefrontOutputs(ctx, client, stackName, hostedZoneID)
			return
		}
	} else {
		// La validación DNS del certificado hace que la creación tarde varios minutos.
		fmt.Printf("Creando stack %v (el certificado tarda unos minutos en validarse)...\n", stackName)
		_, err = client.CreateStack(ctx, &cloudformation.CreateStackInput{
			StackName: &stackName, TemplateBody: &storefrontTemplate,
			Parameters: templateParameters, Capabilities: capabilities,
			OnFailure: cfnTypes.OnFailureDelete,
		})
	}
	if err != nil {
		exitWithDeployError("Error al enviar la plantilla de tiendas a CloudFormation: " + err.Error())
	}

	finalStatus, rootFailureReason := WaitForStackAndPrintEvents(ctx, client, stackName, deployStartedAt)
	if !isSuccessfulStackStatus(finalStatus) {
		message := fmt.Sprintf("El despliegue del stack %v FALLÓ (estado %v).", stackName, finalStatus)
		if len(rootFailureReason) > 0 {
			message += "\nCausa raíz: " + rootFailureReason
		}
		exitWithDeployError(message)
	}

	fmt.Printf("\nStack %v desplegado (%v)\n", stackName, finalStatus)
	syncStorefrontOutputs(ctx, client, stackName, hostedZoneID)
}

// syncStorefrontOutputs escribe los outputs en [storefront] de config.toml: es de ahí de donde los
// leen el backend (backend/core/security.go) y las variables de la Lambda de render.
func syncStorefrontOutputs(ctx context.Context, client *cloudformation.Client, stackName, hostedZoneID string) {
	result, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{StackName: &stackName})
	if err != nil || len(result.Stacks) == 0 {
		fmt.Println("No se pudieron leer los outputs del stack de tiendas: ", err)
		return
	}
	outputs := map[string]string{}
	fmt.Println("\nOutputs del stack:")
	for _, output := range result.Stacks[0].Outputs {
		key, value := aws.ToString(output.OutputKey), aws.ToString(output.OutputValue)
		outputs[key] = value
		fmt.Printf("  %-20v %v\n", key, value)
	}

	configPath := GetConfigPath()
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Println("No se pudo leer", configPath, err)
		return
	}
	updated := upsertTomlSection(string(configBytes), "storefront", [][2]string{
		{"hosted_zone_id", hostedZoneID},
		{"distribution_domain", outputs["DistributionDomain"]},
		{"bucket", outputs["BucketName"]},
	})
	if err := os.WriteFile(configPath, []byte(updated), 0o600); err != nil {
		fmt.Println("No se pudo escribir", configPath, err)
		return
	}
	fmt.Println("\n[storefront] actualizado en", configPath)
	fmt.Println("Publica la configuración en las Lambdas (acción 2) para que el backend y el renderer la lean.")
}

var tomlSectionHeaderPattern = regexp.MustCompile(`^\s*\[`)

// upsertTomlSection fija claves string dentro de [section] por reemplazo de texto, sin
// re-serializar: config.toml se mantiene a mano y sus comentarios importan. Crea la sección al
// final si no existe y añade las claves que falten justo después de la cabecera.
func upsertTomlSection(content, section string, values [][2]string) string {
	lines := strings.Split(content, "\n")
	header := "[" + section + "]"

	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == header {
			start = index
			break
		}
	}
	if start < 0 {
		block := []string{"", header}
		for _, pair := range values {
			block = append(block, fmt.Sprintf("%s = %q", pair[0], pair[1]))
		}
		return strings.TrimRight(content, "\n") + "\n" + strings.Join(block, "\n") + "\n"
	}

	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		if tomlSectionHeaderPattern.MatchString(lines[index]) {
			end = index
			break
		}
	}

	missing := [][2]string{}
	for _, pair := range values {
		keyPattern := regexp.MustCompile(`^(\s*` + regexp.QuoteMeta(pair[0]) + `\s*=\s*)"[^"]*"`)
		found := false
		for index := start + 1; index < end; index++ {
			if keyPattern.MatchString(lines[index]) {
				lines[index] = keyPattern.ReplaceAllString(lines[index], "${1}"+fmt.Sprintf("%q", pair[1]))
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, pair)
		}
	}

	inserted := []string{}
	for _, pair := range missing {
		inserted = append(inserted, fmt.Sprintf("%s = %q", pair[0], pair[1]))
	}
	result := append([]string{}, lines[:start+1]...)
	result = append(result, inserted...)
	result = append(result, lines[start+1:]...)
	return strings.Join(result, "\n")
}
