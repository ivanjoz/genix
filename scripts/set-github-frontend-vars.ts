import { join } from 'node:path'

interface FrontendEndpoint {
	name: string
	route: string
}

interface ProjectCredentials {
	GITHUB_ACCOUNT?: unknown
	LAMBDA_URL?: unknown
	FRONTEND_CDN?: unknown
	ZONE_NAME?: unknown
	ENPOINTS?: unknown
}

const repositoryRoot = join(import.meta.dir, '..')
const credentialsPath = join(repositoryRoot, 'credentials.json')
const isDryRun = process.argv.includes('--dry-run')

const requireNonEmptyString = (credentialsValue: unknown, fieldName: string): string => {
	if (typeof credentialsValue !== 'string' || !credentialsValue.trim()) {
		throw new Error(`credentials.json field ${fieldName} must be a non-empty string`)
	}
	return credentialsValue.trim()
}

const requireHttpUrl = (credentialsValue: unknown, fieldName: string): string => {
	const configuredUrl = requireNonEmptyString(credentialsValue, fieldName)
	const parsedUrl = new URL(configuredUrl)
	if (parsedUrl.protocol !== 'https:' && parsedUrl.protocol !== 'http:') {
		throw new Error(`credentials.json field ${fieldName} must use HTTP or HTTPS`)
	}
	return configuredUrl
}

const requireFrontendEndpoints = (credentialsValue: unknown): FrontendEndpoint[] => {
	if (!Array.isArray(credentialsValue) || credentialsValue.length === 0) {
		throw new Error('credentials.json field ENPOINTS must be a non-empty array')
	}

	return credentialsValue.map((endpointValue, endpointIndex) => {
		if (!endpointValue || typeof endpointValue !== 'object') {
			throw new Error(`credentials.json ENPOINTS[${endpointIndex}] must be an object`)
		}
		const endpointRecord = endpointValue as Record<string, unknown>
		return {
			name: requireNonEmptyString(endpointRecord.name, `ENPOINTS[${endpointIndex}].name`),
			route: requireHttpUrl(endpointRecord.route, `ENPOINTS[${endpointIndex}].route`),
		}
	})
}

const runGitHubCommand = (
	argumentsList: string[],
	operationName: string,
	githubToken?: string,
): string => {
	let commandResult: ReturnType<typeof Bun.spawnSync>
	try {
		commandResult = Bun.spawnSync(['gh', ...argumentsList], {
			cwd: repositoryRoot,
			// GH_TOKEN selects the requested stored account without changing the
			// globally active gh login. Never place the token in command arguments.
			env: githubToken ? { ...process.env, GH_TOKEN: githubToken } : process.env,
			stdout: 'pipe',
			stderr: 'pipe',
		})
	} catch (commandError) {
		throw new Error(`Unable to execute gh while trying to ${operationName}: ${String(commandError)}`)
	}

	if (commandResult.exitCode !== 0) {
		const commandErrorOutput = commandResult.stderr.toString().trim()
		throw new Error(`Unable to ${operationName}${commandErrorOutput ? `: ${commandErrorOutput}` : ''}`)
	}
	return commandResult.stdout.toString().trim()
}

const main = async (): Promise<void> => {
	console.info(`[GitHub Vars] Reading public frontend configuration from ${credentialsPath}`)
	if (!(await Bun.file(credentialsPath).exists())) {
		throw new Error(`credentials.json was not found at ${credentialsPath}`)
	}

	let credentials: ProjectCredentials
	try {
		credentials = JSON.parse(await Bun.file(credentialsPath).text()) as ProjectCredentials
	} catch (parseError) {
		throw new Error(`credentials.json contains invalid JSON: ${String(parseError)}`)
	}

	const frontendEndpoints = requireFrontendEndpoints(credentials.ENPOINTS)
	const githubAccount = requireNonEmptyString(credentials.GITHUB_ACCOUNT, 'GITHUB_ACCOUNT')
	const githubVariables = new Map<string, string>([
		['PUBLIC_LAMBDA_URL', requireHttpUrl(credentials.LAMBDA_URL, 'LAMBDA_URL')],
		['PUBLIC_FRONTEND_CDN', requireHttpUrl(credentials.FRONTEND_CDN, 'FRONTEND_CDN')],
		['PUBLIC_ZONE_NAME', requireNonEmptyString(credentials.ZONE_NAME, 'ZONE_NAME')],
		['PUBLIC_ENDPOINTS', JSON.stringify(frontendEndpoints)],
	])

	const githubToken = runGitHubCommand(
		['auth', 'token', '--hostname', 'github.com', '--user', githubAccount],
		`load the stored GitHub CLI token for ${githubAccount}`,
	)
	const authenticatedAccount = runGitHubCommand(
		['api', 'user', '--jq', '.login'],
		`verify the GitHub account ${githubAccount}`,
		githubToken,
	)
	if (authenticatedAccount.toLowerCase() !== githubAccount.toLowerCase()) {
		throw new Error(
			`GITHUB_ACCOUNT requested ${githubAccount}, but its stored token belongs to ${authenticatedAccount}`,
		)
	}

	const repositoryName = runGitHubCommand(
		['repo', 'view', '--json', 'nameWithOwner', '--jq', '.nameWithOwner'],
		'resolve the current GitHub repository',
		githubToken,
	)
	console.info(`[GitHub Vars] Authenticated account: ${authenticatedAccount}`)
	console.info(`[GitHub Vars] Target repository: ${repositoryName}`)
	console.info(`[GitHub Vars] Validated ${frontendEndpoints.length} API endpoint(s)`)

	for (const [variableName, variableValue] of githubVariables) {
		if (isDryRun) {
			console.info(`[GitHub Vars] Dry run: would set ${variableName}`)
			continue
		}
		runGitHubCommand(
			['variable', 'set', variableName, '--body', variableValue, '--repo', repositoryName],
			`set repository variable ${variableName}`,
			githubToken,
		)
		console.info(`[GitHub Vars] Set ${variableName}`)
	}

	console.info(isDryRun
		? '[GitHub Vars] Dry run completed; GitHub was not modified.'
		: '[GitHub Vars] Frontend repository variables configured successfully.')
}

await main().catch((scriptError) => {
	console.error(`[GitHub Vars] ${scriptError instanceof Error ? scriptError.message : String(scriptError)}`)
	process.exit(1)
})
