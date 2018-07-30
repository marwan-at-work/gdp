# GDP

GDP implements the Go Download Protocol over code hosting APIs. It supports both tagged and untagged modules. 

You can use the gdp implementation programtically or you can use the server in cmd/gdp to test out the functionality against vgo or cmd/go by setting the GOPROXY env var to to http://localhost:8090

Currently GDP supports Github and Bitbucket APIs. 

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

