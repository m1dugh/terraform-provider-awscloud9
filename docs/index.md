---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "awscloud9 Provider"
subcategory: ""
description: |-
  
---

# awscloud9 Provider



## Example Usage

```terraform
# Configuration-based authentication
provider "awscloud9" {
  aws_access_key_id     = "..."
  aws_secret_access_key = "..."
  region                = "us-east-1"
}

# Environment variables-based authentication
provider "awscloud9" {
  region = "us-east-1"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `aws_access_key_id` (String) The AWS access key id, if not provided, extracted from `AWS_ACCESS_KEY_ID` env variable.
- `aws_secret_access_key` (String) The AWS Secret access key, if not provided, extracted from `AWS_SECRET_ACCESS_KEY` env variable.
- `region` (String) The AWS region to use the provider for, if not provided, extracted from `AWS_REGION` env variable.