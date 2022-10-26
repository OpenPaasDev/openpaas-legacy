{{ range $key, $value := .Hosts }}
node "{{$value}}" {
  policy = "write"
}
{{ end }}
