{{ define "packages" }}
    <h1>API reference</h1>

    {{ range .packages }}
        {{ with (index .GoPackages 0 )}}
            {{ with .DocComments }}
                {{ safe (renderComments .) }}
            {{ end }}
        {{ end }}

        Resource Types:

        <ul class="simple">
            {{- range (visibleTypes (sortedTypes .Types)) -}}
                {{ if isExportedType . -}}
                    <li>
                        <a href="{{ linkForType . }}">{{ typeDisplayName . }}</a>
                    </li>
                {{- end }}
            {{- end -}}
        </ul>

        {{ range (visibleTypes (sortedTypes .Types))}}
            {{ template "type" .  }}
        {{ end }}
    {{ end }}

    <div class="admonition note">
        <p class="last">This page was automatically generated with <code>gen-crd-api-reference-docs</code></p>
    </div>
{{ end }}