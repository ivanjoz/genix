// scripts/setup-env.js
import fs from 'fs';
import path from 'path';

export const setupEnv = () => {
  // Keep frontend and backend on the same explicitly selected environment.
  const configuredConfigPath = process.env.GENIX_CONFIG_FILE;
  const configPath = configuredConfigPath
    ? path.resolve(configuredConfigPath)
    : path.resolve(process.cwd(), '..', 'config.toml');
  console.log('📝 Generating .env files from ' + configPath + '...');
  try {
    if (fs.existsSync(configPath)) {
      const config = Bun.TOML.parse(fs.readFileSync(configPath, 'utf-8'));
      // Keep the endpoint selector config available at build time for SvelteKit.
      const serializedPublicEndpoints = JSON.stringify(
        Array.isArray(config.endpoints) ? config.endpoints : []
      );
      // sse_bridge.url is where the agent's event stream lives. It defaults to
      // aws.lambda_url so an install without a bridge keeps talking to the backend
      // directly; point it at the sse_bridge process to make the Lambda
      // deployment able to stream (see PLAN_SSE_BRIDGE.md).
      const envContent = [
        `VITE_PROXY_PORT=${process.env.GENIX_PROXY_PORT || '3572'}`,
        `PUBLIC_LAMBDA_URL=${config.aws?.lambda_url || ''}`,
        `PUBLIC_SSE_BRIDGE_URL=${config.sse_bridge?.url || config.aws?.lambda_url || ''}`,
        `PUBLIC_FRONTEND_CDN=${config.frontend?.cdn_url || ''}`,
        `PUBLIC_ZONE_NAME=${config.frontend?.zone_name || ''}`,
        `PUBLIC_ENDPOINTS=${serializedPublicEndpoints}`
      ].join('\n') + '\n';

      console.log("Seteando Enviroment:")
      console.log(envContent)

      fs.writeFileSync('.env', envContent);
      fs.writeFileSync(path.join('webpage', '.env'), envContent);
      console.log('✅ .env files created successfully');
      return true;
    } else {
      console.warn('⚠️  Config file not found at ' + configPath + ', skipping .env generation');
      return false;
    }
  } catch (error) {
    console.error('❌ Error generating .env files:', error);
    return false;
  }
};
