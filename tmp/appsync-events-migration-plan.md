# AppSync Events API Migration Plan

## Current State
- **Deployment**: `p2p/deploy` uses AWS CDK Go v2.114.1
- **Implementation**: GraphQL API with `awsappsync.NewGraphqlApi`
- **Schema**: `schema.graphql` defines Signal type, Query, Mutation, Subscription
- **Authorization**: API key-based
- **Purpose**: P2P signaling for real-time communication

## Problem
The current implementation uses GraphQL, which is designed for structured data queries and mutations. For P2P real-time signaling, a simpler pub/sub model is more appropriate and cost-effective.

## Solution: AppSync Events API
Convert to AWS AppSync Events API, which provides:
- WebSocket-based real-time pub/sub
- No GraphQL schema required
- Better suited for high-scale broadcasting
- Simpler client-side integration

## Implementation Strategy

### Phase 1: Research and Feasibility Check
1. **Verify CDK Support**: Check if AWS CDK Go v2.114.1 supports AppSync Events API constructs
2. **Alternative Approaches**: 
   - If CDK supports: Use CDK constructs
   - If CDK doesn't support: Use CloudFormation custom resource or AWS SDK within CDK
3. **Identify Required Changes**:
   - Remove: GraphQL schema, resolvers, data sources
   - Add: Event API configuration, channel namespaces, authorization

### Phase 2: Code Changes

#### File: `deploy.go`

**Remove:**
- `awsappsync.NewGraphqlApi` - Replace with Event API
- `api.AddNoneDataSource` - Not needed for Event API
- `dataSource.CreateResolver` - No resolvers in Event API
- GraphQL schema file reference
- Mutation/Subscription resolvers

**Add:**
- Event API creation using appropriate construct
- Channel namespace configuration (e.g., `p2p-signal`)
- Authorization configuration (API key or other)
- WebSocket endpoint output
- Authorization header/key output

**Pseudo-code structure:**
```go
// Create AppSync Events API
eventApi := awsappsync.NewCfnGraphqlApi(stack, jsii.String("SignalingEventApi"), &awsappsync.CfnGraphqlApiProps{
    Name: jsii.String(cfg.GetStackName() + "-events-api"),
    ApiType: jsii.String("EVENT"), // Key property for Event API
    AuthorizationConfig: &awsappsync.CfnGraphqlApi_AuthorizationConfigProperty{
        DefaultAuthorization: &awsappsync.CfnGraphqlApi_AuthorizationConfigProperty{
            AuthorizationType: jsii.String("API_KEY"),
        },
    },
    // Optional: Event API configurations
    // EventConfig: ... (if supported)
})

// Create API key for Event API
apiKey := awsappsync.NewCfnApiKey(stack, jsii.String("EventApiKey"), &awsappsync.CfnApiKeyProps{
    ApiId: eventApi.AttrApiId(),
})

// Outputs
awscdk.NewCfnOutput(stack, jsii.String("WebSocketUrl"), &awscdk.CfnOutputProps{
    Value: eventApi.AttrWebSocketUrl(),
})

awscdk.NewCfnOutput(stack, jsii.String("ApiKey"), &awscdk.CfnOutputProps{
    Value: apiKey.AttrApiKey(),
})
```

#### File: `schema.graphql`
**Action**: DELETE - No longer needed for Event API

### Phase 3: Client Integration Updates

The client-side code will need to change from GraphQL subscriptions to WebSocket pub/sub:

**Old (GraphQL):**
```javascript
const subscription = gql`
  subscription OnSignal($to: String!) {
    onSignal(to: $to) {
      from
      to
      action
      data
    }
  }
`;
```

**New (Event API):**
```javascript
// Connect to WebSocket
const ws = new WebSocket('wss://api-id.appsync-api.region.amazonaws.com/event');

// Subscribe to channel
ws.send(JSON.stringify({
  action: 'subscribe',
  channel: 'p2p-signal/to-peer-id'
}));

// Publish to channel
ws.send(JSON.stringify({
  action: 'publish',
  channel: 'p2p-signal/to-peer-id',
  data: { from, to, action, data }
}));
```

### Phase 4: Testing
1. **Deploy**: Run `cdk deploy` to create the Event API
2. **Verify Outputs**: Check that WebSocket URL and API key are generated
3. **Test Connectivity**: Connect to WebSocket endpoint
4. **Test Pub/Sub**: Verify publish and subscribe operations work correctly
5. **Load Testing**: Test with multiple concurrent connections

## Alternative Approach (If CDK Doesn't Support Event API)

If AWS CDK Go v2.114.1 doesn't have native AppSync Events API support, use CloudFormation:

```go
// Use CloudFormation custom resource
eventApi := awscdk.NewCfnResource(stack, jsii.String("EventApi"), &awscdk.CfnResourceProps{
    Type: jsii.String("AWS::AppSync::GraphQLApi"),
    Properties: map[string]interface{}{
        "Name":           jsii.String(cfg.GetStackName() + "-events-api"),
        "ApiType":        jsii.String("EVENT"),
        "AuthenticationType": jsii.String("API_KEY"),
        // Additional Event API properties
    },
})
```

## Benefits of Migration

1. **Simplicity**: No GraphQL schema or resolvers to manage
2. **Cost**: Potentially lower cost for simple pub/sub use cases
3. **Performance**: Optimized for real-time event delivery
4. **Scalability**: Designed for high-scale WebSocket connections
5. **Code Reduction**: Simpler deployment code

## Risks and Mitigations

1. **Risk**: CDK Go may not support AppSync Events API
   - **Mitigation**: Use CloudFormation custom resource or upgrade CDK version

2. **Risk**: Breaking change for existing clients
   - **Mitigation**: Plan for client migration, potentially support both APIs during transition

3. **Risk**: Authorization model differences
   - **Mitigation**: Test API key authorization with Event API

## Success Criteria

- [ ] Event API deploys successfully via CDK
- [ ] WebSocket endpoint is accessible and functional
- [ ] Pub/Sub operations work correctly
- [ ] Client code successfully connects and communicates
- [ ] No GraphQL schema or resolver code remains
- [ ] Deployment code is simplified

## Implementation Priority

1. **High**: Convert deploy.go to use Event API
2. **Medium**: Remove schema.graphql and GraphQL-related code
3. **Medium**: Update client-side integration (may be separate task)
4. **Low**: Optimize channel namespacing and authorization

## Notes

- AppSync Events API is a relatively new AWS feature (released in late 2023)
- The API type "EVENT" is the key differentiator from standard GraphQL APIs
