path "secret/*" { #some path in secrets
    capabilities = ["read"]
}
//vault policy write backend access.hcl