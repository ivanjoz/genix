Connect your app
Connect a client
You can use any WebSocket client to connect to your AppSync Event API. We recommend using the AWS Amplify client to get started.

On your local machine, in your project folder, install the Amplify library:

npm install aws-amplify

Download your configuration file and place it in your project folder.

Download config
In your code, import the Amplify Library and your configuration to set up Amplify.

import { Amplify } from 'aws-amplify';
import { events } from 'aws-amplify/data';
import config from './amplify_outputs.json';

Amplify.configure(config);

Alternatively, use the snippet with your API configuration to set up Amplify.

import { Amplify } from 'aws-amplify';
import { events } from 'aws-amplify/data';

Amplify.configure({
  "API": {
    "Events": {
      "endpoint": "https://2v7tfrfxenfpvgf356vcuw3iie.appsync-api.us-east-1.amazonaws.com/event",
      "region": "us-east-1",
      "defaultAuthMode": "apiKey",
      "apiKey": "da2-*********************"
    }
  }
});

Your client is now configured. To connect to your channel and subscribe:

const channel = await events.connect('/default/test');
channel.subscribe({
  next: (data) => {
    console.log('received', data);
  },
  error: (err) => console.error('error', err),
});

To publish events over the HTTP endpoint:

// Single event
await events.post('/default/test', {some: 'data'});

// Multiple events
await events.post('/default/test', [{some: 'data'}, {and: 'data'}, {more: 'data'}]);
