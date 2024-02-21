# appscode-kubebuilder

# download kubebuilder and install locally.
curl -L -o kubebuilder "https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)"

chmod +x kubebuilder

sudo mv kubebuilder /usr/local/bin/

[//]: # (mkdir -p ~/projects/guestbook)

[//]: # (cd ~/projects/guestbook)
kubebuilder init --domain "github.com/sheikh-arman" --repo "github.com/sheikh-arman/appscode-kubebuilder"

