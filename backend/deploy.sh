#!/bin/bash
AWS_PROFILE="hortifrut_prod"
FUNCTION_NAME_GO="hf-smartberry-etl"
ENVIROMENT_NAME="PRD"

echo "Obteniendo los últimos cambios del repositorio (GIT PULL)..."

git pull
echo "Usando AWS Profile: $AWS_PROFILE"

#PUBLICAR BACKEND-GO
echo "=== PUBLICANDO BACKEND ==="

if [[ $ENVIROMENT == "1" ]]; then #DEV-HORTIFRUT
    FUNCTION_NAME_GO=$FUNCTION_NAME_GO_DEV_HF

elif [[ $ENVIROMENT == "2" ]]; then #PROD-HORTIFRUT
    FUNCTION_NAME_GO=$FUNCTION_NAME_GO_PRD_HF

elif [[ $ENVIROMENT == "3" ]]; then #PROD-YAWI
    FUNCTION_NAME_GO=$FUNCTION_NAME_GO_PROD

fi

echo "Enviado código de funcion: $FUNCTION_NAME_GO | profile: $AWS_PROFILE"

echo "Generando compilado..."
echo "Ejecutando:  GOOS=linux GOARCH=arm64 go build -ldflags -X main.ENVIROMENT=$ENVIROMENT_NAME -o .aws-sam/build/main"

# BACKEND_BUILD_TAGS is required on every shipping build, not optional size tuning: both tags drop
# a package that calls reflect.Value.MethodByName with a runtime string, which sets the linker's
# global reflectSeen flag and retains every exported method of every reachable type. Omitting them
# costs ~12 MB. cloud/main.go, scripts/deploy_vps.go and .github/workflows/release-binaries.yml
# carry the same list; backend/thirdparty/patches_test.go keeps them in sync.
BACKEND_BUILD_TAGS="lambda.norpc,grpcnotrace"

if [[ "$OSTYPE" == "linux-gnu"* ]]; then # comando para LINUX
    GOOS=linux GOARCH=arm64 /usr/local/go/bin/go build -tags "$BACKEND_BUILD_TAGS" -ldflags "-s -w" -o .aws-sam/build/main
else # comando para WINDOWS
    GOOS=linux GOARCH=arm64 go build -tags "$BACKEND_BUILD_TAGS" -ldflags "-s -w -X main.ENVIROMENT=$ENVIROMENT_NAME" -o .aws-sam/build/main
fi

# crea el zip para ser subido a aws-lambda
# para ello hay que instalar build-lambda-zip (librería en go)
# go install github.com/aws/aws-lambda-go/cmd/build-lambda-zip
echo "Generando archivo .zip..."
if [[ "$OSTYPE" == "linux-gnu"* ]]; then # comando para LINUX
    ~/go/bin/build-lambda-zip -output .aws-sam/build/main.zip .aws-sam/build/main
else # comando para WINDOWS
    ~/go/bin/build-lambda-zip -output .aws-sam/build/main.zip .aws-sam/build/main
fi
# se necesita aquí tener instalado el AWS-CLI
echo "Enviando código Lambda..."
aws --profile=$AWS_PROFILE lambda update-function-code --function-name $FUNCTION_NAME_GO --zip-file fileb://.aws-sam/build/main.zip

echo "El deploy backend-golang finalizado!"
read
kill -9 $PPID
