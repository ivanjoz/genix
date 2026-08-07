// scripts/setup-env.js
import fs from 'fs';
import path from 'path';

export const setupEnv = () => {
  // Keep frontend and backend on the same explicitly selected environment.
  const configuredCredentialsPath = process.env.GENIX_CREDENTIALS_FILE;
  const credentialsPath = configuredCredentialsPath
    ? path.resolve(configuredCredentialsPath)
    : path.resolve(process.cwd(), '..', 'credentials.json');
  console.log('📝 Generating .env files from ' + credentialsPath + '...');
  try {
    if (fs.existsSync(credentialsPath)) {
      const credentials = JSON.parse(fs.readFileSync(credentialsPath, 'utf-8'));
      // Keep the endpoint selector config available at build time for SvelteKit.
      const serializedPublicEndpoints = JSON.stringify(
        Array.isArray(credentials.ENPOINTS) ? credentials.ENPOINTS : []
      );
      // SSE_BRIDGE_URL is where the agent's event stream lives. It defaults to
      // LAMBDA_URL so an install without a bridge keeps talking to the backend
      // directly; point it at the sse_bridge process to make the Lambda
      // deployment able to stream (see PLAN_SSE_BRIDGE.md).
      const envContent = [
        `VITE_PROXY_PORT=${process.env.GENIX_PROXY_PORT || '3572'}`,
        `PUBLIC_LAMBDA_URL=${credentials.LAMBDA_URL || ''}`,
        `PUBLIC_SSE_BRIDGE_URL=${credentials.SSE_BRIDGE_URL || credentials.LAMBDA_URL || ''}`,
        `PUBLIC_FRONTEND_CDN=${credentials.FRONTEND_CDN || ''}`,
        `PUBLIC_ZONE_NAME=${credentials.ZONE_NAME || ''}`,
        `PUBLIC_ENDPOINTS=${serializedPublicEndpoints}`
      ].join('\n') + '\n';
      
      console.log("Seteando Enviroment:")
      console.log(envContent)

      fs.writeFileSync('.env', envContent);
      fs.writeFileSync(path.join('webpage', '.env'), envContent);
      console.log('✅ .env files created successfully');
      return true;
    } else {
      console.warn('⚠️  Credentials file not found at ' + credentialsPath + ', skipping .env generation');
      return false;
    }
  } catch (error) {
    console.error('❌ Error generating .env files:', error);
    return false;
  }
};
