# Qorvi Production Terraform Workspace

This directory must not contain committed Terraform state, tfvars, or runtime
secrets.

Production runtime secrets are managed through GCP Secret Manager and rendered
on the VM into `/opt/qorvi/app/.env.wallet-secrets`. Keep provider keys,
database passwords, Clerk secrets, and API tokens out of Terraform variables,
state files, startup scripts, and repository history.

If local Terraform state or `terraform.tfvars` is needed for emergency
operations, keep it outside the repository workspace with filesystem
permissions restricted to the operator account.
