# GDP

GDP implements the [Go Download Protocol](https://github.com/golang/go/wiki/Modules) over code hosting APIs. It supports both tagged and untagged modules. 

You can use the gdp implementation programtically or you can use the server in cmd/gdp to test out the functionality against vgo or cmd/go by setting the GOPROXY env var to to http://localhost:8090

Currently GDP supports Github, Bitbucket and Gopkg.in. 

### Example

Create a test a repo outside of GOPATH with the following two files


```golang
// main.go

package main

import _ "github.com/pkg/errors"

func main() {}
```

```
// go.mod
module github.com/myuser/mytest
```

Then run `GOPROXY=http://localhost:8090 go build`


### Options

You should always pass -token to cmd/gdp to get around GitHub's rate limiting. 

If you are building a package that's none of the APIs mentioned above (such as golang.org/x/...), the proxy returns 
a 404. You can alternatively give cmd/gdp a -redirect flag so that you can redirect to another GOPROXY such as Athens.