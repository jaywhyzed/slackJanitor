# README

## Deploy Command

```sh
$ gcloud functions deploy create_channel \
--trigger-http \
--runtime=go116 \
--region=us-central1 \
--entry-point=CreateChannelHandler \
--set-secrets 'VC_URL=MEET_URL:latest' \
--set-secrets 'SLACK_BOT_USER_TOKEN=SLACK_BOT_USER_TOKEN:latest' \
--set-secrets 'VC_CALL_ID=MEET_CALL_ID:latest'
```

### Update Command

```sh
$ gcloud functions deploy create_channel
```

## Test Deployed Version with

```sh
 curl -H "Authorization: bearer $(gcloud auth print-identity-token)" $URL
```
