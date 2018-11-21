# Git Store
Git abstraction layer written in Go, mainly for use in Kubernetes Controllers.

Git Store is based on [Go Git](https://github.com/src-d/go-git) and provides convenience methods
for cloning, fetching and checking out repository references and accessing file contents within a repository.

## Usage

To get a slice of all yaml and json file from a repository at a given reference:

```
store := gitstore.newRepoStore()

repo, err := store.Get(&gistore.RepoRef{
	URL: 		url,
	PrivateKey:	privateKey,
})

err = repo.Checkout(ref)
lastUpdated, err := repo.LastUpdated()

globbedSubPath := strings.TrimPrefix(gt.Spec.SubPath, "/") + "{**/*,*}.{yaml,yml,json}"
files, err := repo.GetAllFiles(globbedSubPath, true)

for file := range files {
	doStuffWith(file.Contents()
}
```

## Communication

* Found a bug? Please open an issue.
* Have a feature request. Please open an issue.
* If you want to contribute, please submit a pull request

## Contributing
Please see our [Contributing](CONTRIBUTING.md) guidelines.

## License
This project is licensed under Apache 2.0 and a copy of the license is available [here](LICENSE).
