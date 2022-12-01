go build -gcflags="all=-N -l" -o go-whatsapp-proxy
#dlv --listen=:2345 --headless=true --api-version=2 exec ./go-whatsapp-proxy