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


## WORKING EXAMPLE OFFICIAL

Request URL
https://2v7tfrfxenfpvgf356vcuw3iie.appsync-api.us-east-1.amazonaws.com/event
Request Method
POST
Status Code
200 OK
Remote Address
108.158.104.92:443
Referrer Policy
strict-origin-when-cross-origin
access-control-allow-origin
*
access-control-expose-headers
x-amzn-RequestId,x-amzn-ErrorType,x-amz-user-agent,x-amzn-ErrorMessage,Date,x-amz-schema-version
content-length
156
content-type
application/json; charset=UTF-8
date
Thu, 29 Jan 2026 05:32:35 GMT
via
1.1 17ce9cbcbc686d5320c94e5563c8e4e6.cloudfront.net (CloudFront)
x-amz-cf-id
pq0dnKo1UmE4KidmV6KqunlDVvqHBI-GTbHEpOVAxhb8b4d2asqJrQ==
x-amz-cf-pop
LIM50-P1
x-amzn-appsync-tokensconsumed
1
x-amzn-remapped-content-type
application/json; charset=UTF-8
x-amzn-requestid
9909d997-9d40-4770-95b3-4feabd5ce482
x-cache
Miss from cloudfront
:authority
2v7tfrfxenfpvgf356vcuw3iie.appsync-api.us-east-1.amazonaws.com
:method
POST
:path
/event
:scheme
https
accept
*/*
accept-encoding
gzip, deflate, br, zstd
accept-language
es-US,es;q=0.9,en-US;q=0.8,en;q=0.7,es-419;q=0.6,fr;q=0.5
cache-control
no-cache
content-length
99
content-type
application/json
origin
https://us-east-1.console.aws.amazon.com
pragma
no-cache
priority
u=1, i
referer
https://us-east-1.console.aws.amazon.com/
sec-ch-ua
"Google Chrome";v="143", "Chromium";v="143", "Not A(Brand";v="24"
sec-ch-ua-mobile
?0
sec-ch-ua-platform
"Linux"
sec-fetch-dest
empty
sec-fetch-mode
cors
sec-fetch-site
cross-site
user-agent
Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36
x-amz-date
20260129T053234Z
x-amz-user-agent
AWS-Console-AppSync/
x-api-key
da2-czyagszanfbptkl57l3t45xmo4


## PAYLOAD

{"channel":"genix-bridge/channel","events":["{\"event_1\":\"data_1\"}","{\"event_2\":\"data_2\"}"]}

## WEBSOCKET REAL EXAMPLE

Request URL
wss://2v7tfrfxenfpvgf356vcuw3iie.appsync-realtime-api.us-east-1.amazonaws.com/event/realtime
Request Method
GET
Status Code
101 Switching Protocols
connection
Upgrade
sec-websocket-accept
C0+NB/ver996GA1KkRuluIGa5yg=
sec-websocket-protocol
aws-appsync-event-ws
upgrade
websocket
x-amzn-requestid
4628a461-80c7-40c0-b5bc-b6f1ec7645d7
accept-encoding
gzip, deflate, br, zstd
accept-language
es-US,es;q=0.9,en-US;q=0.8,en;q=0.7,es-419;q=0.6,fr;q=0.5
cache-control
no-cache
connection
Upgrade
host
2v7tfrfxenfpvgf356vcuw3iie.appsync-realtime-api.us-east-1.amazonaws.com
origin
https://us-east-1.console.aws.amazon.com
pragma
no-cache
sec-websocket-extensions
permessage-deflate; client_max_window_bits
sec-websocket-key
M9aA+sEjAzR82ashUfa5rQ==
sec-websocket-protocol
aws-appsync-event-ws, header-eyJob3N0IjoiMnY3dGZyZnhlbmZwdmdmMzU2dmN1dzNpaWUuYXBwc3luYy1hcGkudXMtZWFzdC0xLmFtYXpvbmF3cy5jb20iLCJ4LWFtei1kYXRlIjoiMjAyNjAxMjlUMDUzNDU0WiIsIngtYXBpLWtleSI6ImRhMi1jenlhZ3N6YW5mYnB0a2w1N2wzdDQ1eG1vNCJ9
sec-websocket-version
13
upgrade
websocket
user-agent
Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36
