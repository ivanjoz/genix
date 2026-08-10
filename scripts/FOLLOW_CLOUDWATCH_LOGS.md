# Follow CloudWatch Logs

Sigue en tiempo real el grupo de logs de la Lambda principal del backend:

```text
/aws/lambda/<app_name>-backend
```

`app_name`, `aws.profile` y `aws.region` se leen del archivo elegido en la pestaña
**Environment** de `deploy.sh`. El grupo coincide con `BackendLogGroup` en
`cloud/template.yml`.

## Uso

Desde la interfaz, seleccione **Scripts → Servidores → Follow Cloudwatch Logs**. También se
puede ejecutar sin abrir el TUI:

```bash
GENIX_CONFIG_FILE=config.1.toml ./deploy.sh follow_cloudwatch_logs
```

El comando ejecutado es equivalente a:

```bash
aws --profile <aws.profile> --region <aws.region> logs tail \
  /aws/lambda/<app_name>-backend --follow
```

Termine el seguimiento con `Ctrl+C`. Requiere AWS CLI y permisos de lectura de CloudWatch
Logs sobre el grupo.
