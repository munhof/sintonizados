# Contenedores

```sh
podman build -t localhost/sintonizados:dev .
podman run --rm --name sintonizados -p 127.0.0.1:8080:8080 \
  --env-file .env.example --read-only --cap-drop=ALL \
  --security-opt=no-new-privileges localhost/sintonizados:dev
```

El `Containerfile` compila servidor y emisor estáticos con Go 1.26.1, usa scratch,
CAs y usuario no root. No depende de Compose ni Kubernetes. `.containerignore`
evita incluir credenciales y estado de Git; COPY selecciona sólo código Go.

El wrapper ejecuta Go con `--userns=keep-id`; monta código con `:Z` y usa volúmenes
`sintonizados-go-mod` y `sintonizados-go-build`. Smithy CLI 1.73.0 se descarga dentro
de una imagen Java, verificando el SHA256 publicado, y admite amd64/arm64. Se comprobó
amd64. `api/build` es temporal; `api/generated/openapi` se versiona.

Los validadores Python se instalan dentro de contenedores efímeros. Las versiones
directas están fijadas; las imágenes base usan tags y las dependencias transitivas
Python no están fijadas con hashes. Para releases reproducibles bit a bit, fijar
digests y locks como trabajo posterior, sin afirmar esa propiedad hoy.

`./scripts/dev contract` inicia un servidor aislado sin publicar puertos y ejecuta
un cliente de validación en su namespace de red. Comprueba tipos/campos de respuestas
con JSON Schema 2020-12 y todas las operaciones modeladas; luego elimina el contenedor.
