# Instructions to get local Portal working

I found while trying to get the Portal running locally that some of the environment variables point to outdated Entra ID stuff that's no longer working. I ended up creating my own Entra ID app for faking the Portal's identity in local dev and also updating some other stuff. Steps below.

## Env setup

1. In the Azure Portal (Red Hat dev sub), go to Entra Id and choose to Add an App Registration
    a. This will be used as the Portal's identity
    b. Name it whatever you want - it's for your personal use in your local dev env (the shared one isn't working and I didn't feel like updating the shared one right now). I could share mine with you, but it's more convenient for you to create your own because you'll have full administrative privileges over it in case something isn't working and you need to make changes.
    c. For the optional "Redirect URI", choose type "Web" and put in "https://localhost:8444/callback" for the URI. This tells Entra ID what endpoint on the Portal it should redirect us to after we've successfully authenticated to Entra ID
2. Browse to your new App Registration, then go to Manage -> Token configuration. Choose "Add groups claim", choose "All Groups", and then save the changes.
    a. This tells Entra ID to include our Entra ID group memberships in the token it gives to the Portal after we go through Entra ID auth
3. Grab the client ID of your app registration and add it to your local RP env file: `export AZURE_PORTAL_CLIENT_ID=<your-client-id>`
    a. Make sure this goes **after** the line where env sources secrets/env - we want to overwrite the value that comes from secrets/env
4. In your local RP directory, run `go run ./hack/genkey -client portal-client` to generate a certificate for the new app registration to use. Then run `az ad app credential reset --id "$AZURE_PORTAL_CLIENT_ID" --cert "$(base64 -w0 <portal-client.crt)" >/dev/null` to associate it with the app registration.
5. In the same spot in your env file where you added the Portal identity's client ID, add these other two lines to override some more values from secrets/env:
    a. `export AZURE_PORTAL_ACCESS_GROUP_IDS=840513b6-3ae6-47ed-8f5a-ac429e819c66`
    b. `export AZURE_PORTAL_ELEVATED_GROUP_IDS=840513b6-3ae6-47ed-8f5a-ac429e819c66`
    c. The Portal uses these to decide which Entra ID groups are allowed to access the Portal. The value in secrets/env doesn't seem to exist anymore, so we're overriding them here with a different group that contains the whole ARO team
6. Make sure you're on this branch when you try running the Portal. I added some hacky stuff to `cmd/aro/portal.go` to load the certificate from the local file instead of trying to get it from key vault.

Most of these steps were adapted from `docs/prepare-a-shared-rp-development-environment.md`.

## Running the Portal and trying out changes

See `docs/admin-portal.md`.

To run the Portal without having made any changes, see the heading `Running Portal Served from development RP`

To try out changes to the Portal, see the heading `Developing`
