# Migration Plan: WebRTC Signaling via AWS AppSync

This document outlines the strategy for replacing the current WebSocket/Lambda signaling infrastructure with a serverless GraphQL pub/sub model using AWS AppSync.

## 1. Objective
Replace the stateful WebSocket management (currently handled by a Lambda function and API Gateway) with AWS AppSync Subscriptions. This simplifies the architecture by leveraging AppSync's built-in real-time capabilities, removing the need for the `signaling_lambda` and the logic for tracking connection IDs.

## 2. New Architecture
- **AWS AppSync**: Acts as the signaling hub.
- **Homelab Server**: Subscribes to a specific `topic` (e.g., its own ID) via AppSync GraphQL Subscriptions.
- **Frontend**: Sends WebRTC offers/candidates via AppSync GraphQL Mutations.
- **Authentication**: API Key (initial phase) or AWS IAM/OIDC.

## 3. GraphQL Schema
```graphql
type Signal {
  from: String!
  to: String!
  action: String!
  data: String! # JSON string containing WebRTC SDP or ICE candidates
}

type Query {
  # Required by AppSync, but not used for signaling
  getSignal(id: ID!): Signal
}

type Mutation {
  sendSignal(to: String!, action: String!, data: String!): Signal
}

type Subscription {
  onSignal(to: String!): Signal
    @aws_subscribe(mutations: ["sendSignal"])
}

schema {
  query: Query
  mutation: Mutation
  subscription: Subscription
}
```

## 4. Implementation Steps

### Step 1: Update Infrastructure (CDK)
**File**: `p2p/deploy/deploy.go`
- **Remove**: `SignalingLambda`, `SignalingApi` (WebSocket), and associated permissions.
- **Add**: 
  - `awscdkappsync.GraphqlApi`
  - `awscdkappsync.CfnGraphQLSchema` (using the schema above)
  - `awscdkappsync.NoneDataSource` (local resolver for pub/sub)
  - `awscdkappsync.CfnResolver` for `sendSignal` mutation.
- **Output**: AppSync Endpoint URL and API Key.

### Step 2: Update Homelab Server (Go)
**File**: `p2p/homelab_server/main.go`
- **Remove**: `gorilla/websocket` client logic.
- **Add**: AppSync GraphQL over WebSocket client.
  - Subscribe to `onSignal(to: "genix_bridge")`.
  - Handle incoming signals (offers) and send responses (answers) via mutations.
- **Update**: Remove `updateLambdaConfig` since `LAPTOP_ID` tracking in Lambda environment variables is no longer needed.

### Step 3: Update Frontend
**File**: `frontend/pkg-core/lib/wss-webrtc.ts`
- **Remove**: `WebSocket` logic.
- **Add**: AppSync client integration.
  - Use `Mutation` to send signals.
  - Use `Subscription` to receive signals from the homelab.
- **Update**: `WSSWebRTC` class to use GraphQL operations instead of raw WebSocket messages.

### Step 4: Cleanup
- **Delete**: `p2p/signaling_lambda/` folder.
- **Delete**: `p2p/update-lambda.go`.
- **Delete**: `p2p/build-lambda.sh`.
- **Update**: `p2p/deploy.sh` to remove Lambda build steps.
- **Update**: `p2p/.gitignore` and `README.md`.

## 5. Benefits
- **Zero Compute**: No Lambda execution for routing messages.
- **Native Scalability**: AppSync handles millions of connections and message delivery out of the box.
- **Simplified State**: No need to manually track `connectionId` or update environment variables.
- **Typed Protocol**: GraphQL provides a clear contract between frontend and backend.
