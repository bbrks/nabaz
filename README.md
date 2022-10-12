![Test](https://github.com/nabaz-io/nabaz/actions/workflows/test-nabaz.yaml/badge.svg) [![Go Reference](https://pkg.go.dev/badge/github.com/nabaz-io/nabaz.svg)](https://pkg.go.dev/github.com/nabaz-io/nabaz) [![License][license-image]][license-url]

# nabaz: The change based test runner
Hate waiting for tests?    
Reduce up to **98%** of test run time by skipping tests unaffected by code.

**Under the hood:**
Gathers code coverage for individual tests, compares new code to code coverage.

![nabaz installation](https://raw.githubusercontent.com/nabaz-io/nabaz/main/docs/goinstall.gif)

Run it **locally** like so:

```bash
CMDLINE="pytest -v"
nabaz test --cmdline $CMDLINE ./
```
_**Note:** [Contact us](#contact-us) for early access to **CI/CD** version (remote storage, integration, system tests support)._

---
## Installation

 ### **Linux** 🐧
```bash
# Ubuntu
wget -qO- https://nabaz.jfrog.io/artifactory/api/security/keypair/nabazgpg/public | apt-key add -
echo "deb [arch=amd64] https://nabaz.jfrog.io/artifactory/nabaz-debian-local stable main" >> /etc/apt/sources.list
sudo apt update
sudo apt install -y nabaz

# Debian
wget https://nabaz.jfrog.io/artifactory/nabaz-debian-local/pool/stable/nabaz-0.0-amd64.deb -O nabaz.deb
sudo dpkg -i ./nabaz.deb
```
  
### **With `go install`**

You can install `nabaz` using the `go install` command:

```bash
# make sure PATH is set up
go install github.com/nabaz-io/nabaz/cmd/nabaz@latest

cd $GOPATH/src/github.com/nabaz-io

```
You can then use the `nabaz` command, provided that your Go `bin` directory is added to your system path.

### **From source**
```
# Install nabaz binary.
go install github.com/nabaz-io/nabaz@latest
cd nabaz
go build -o ./bin/nabaz cmd/nabaz
sudo mv bin/* /usr/bin/
sudo chmod +x /usr/bin/nabaz


# Install our modified go (for running
cd $GOPATH/github.com/nabaz-io
git clone https://github.com/nabaz-io/go
cd go/src
./make.bash
mv $GOPATH/src/github.com/nabaz-io/go /usr/local/nabaz-go
```

---
# Language Support
## Languages
- [x] Python 🐍
- [x] Go 
- [ ] Java (Coming soon)
- [ ] .NET/C# (Coming soon)
- [ ] Javascript (TBD)
- [ ] C/C++ (not planned currently)
- [ ] [Request here](https://github.com/nabaz-io/nabaz/issues/new?assignees=&labels=&template=feature_request.md&title=)
## Frameworks
- [x] pytest
- [x] go test
- [ ] JUnit
- [ ] XUnit
- [ ] Cypress
- [ ] [Request here](https://github.com/nabaz-io/nabaz/issues/new?assignees=&labels=&template=feature_request.md&title=)

---
## Building

```bash
 go build -o nabaz ./cmd/nabaz
 ```

## Contact us!
at hello@nabaz.io.
## License

Licensed under the [MIT license](LICENSE.md).

[license-image]: https://img.shields.io/:license-mit-blue.svg
[license-url]: LICENSE.md
