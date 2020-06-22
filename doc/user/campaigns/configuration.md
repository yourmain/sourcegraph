# Configuration

## Disabling campaigns

Campaigns are enabled by default. In order to disable the feature for all users, a site-admin of your Sourcegraph instance must disable it in the site configuration settings e.g. `sourcegraph.example.com/site-admin/configuration`

```json
{
  "campaigns.enabled": false
}
```

## Code host configuration

When using campaigns with repositories hosted on GitHub, make sure that the GitHub connection configured in Sourcegraph uses a token with the [required token scopes](../../admin/external_service/github.md#github-api-token-and-access). Otherwise campaigns won't be able to create changesets (pull requests) on the configured GitHub instance and sync them back to Sourcegraph.

The user associated with the token also needs to have write-access to the repository in order to create changesets when creating campaigns.
