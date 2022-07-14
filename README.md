# smart-cd-operator
```
go init
operator-sdk init --domain org.smart --repo github.com/DevYoungHulk/smart-cd-operator
operator-sdk create api --version=v1alpha1 --kind=Canary --group cd

go mod tidy
go mod vendor

make build
make install

make run

make docker-build
make docker-push
make deploy
```