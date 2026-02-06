# go-tdms

[![CI](https://github.com/drewsilcock/go-tdms/actions/workflows/ci.yaml/badge.svg)](https://github.com/drewsilcock/go-tdms/actions/workflows/ci.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/drewsilcock/go-tdms.svg)](https://pkg.go.dev/github.com/drewsilcock/go-tdms)

This is a pure Go no-dependency file parser for the Technical Data Management Streaming (TDMS) format used by National Instruments (NI) software such as LabVIEW.

## Usage

Install with:

```shell
go get -u github.com/drewsilcock/go-tdms
```

Open and explore TDMS files like so:

```go
file, err := tdms.Open("data.tdms")
if err != nil {
	log.Fatal(err)
}
defer file.Close()

for _, group := range file.Groups {
	for _, channel := range group.Channels {
		// Iterate through individual values (uses batching internally).
		for value, err := range channel.ReadDataAsFloat64() {
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(value)
		}

		// Iterate through batches of values.
		for batch, err := range channel.ReadDataAsFloat64Batch() {
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(batch)
		}

		// Batch size is configurable (both for individual value streamer and
		// batch streamer)
		for batch, err := range channel.ReadDataAsFloat64Batch(tdms.BatchSize(1024)) {
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(batch)
		}

		// Read all values into a single slice
		values, err := channel.ReadDataAsFloat64All() {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(values)
	}
}

// Access property value string using the `As[Type]()` methods.
authorProp := file.Properties["Author"]
author, err := authorProp.AsString()
if err != nil {
	log.Fatal(err)
}
fmt.Println("Why, this TDMS file was written by none other than ", author)
```

## Status

As of February 2026, this is being actively maintained but has not been battled-tested.

| Feature                                     | Status |
|---------------------------------------------|--------|
| Reading TDMS file full data files           | ☑️     |
| Reading TDMS file index files               | ☑️     |
| Reading properties from file objects        | ☑️     |
| Reading data from channels                  | ☑️     |
| Streaming data from channels                | ☑️     |
| Extended precision floating point data type | ☑️     |
| Timestamp floating point data type          | ☑️     |
| Complex floating point data types           | ☑️     |
| Multi-chunk segments                        | ☑️     |
| Data interleaving                           | ☑️     |
| Data scaling                                | □      |
| DAQmx data and scalers                      | □      |
| Fixed point numerics                        | □      |

### Future work

#### Data scaling and DAQmx

I need to read up more about how the data scaling works and what exactly DAQmx is. The official documentation on this is either very confusing or non-existent, so the best source is information is usually the npTDMS source code.

#### Fixed point numerics

The official documentation does not provide any detail on what format the fixed point numerics are stored on disk with, and I cannot find any examples of TDMS files with fixed point numerics on the internet, so until I can find more information this is going to remain unimplemented.

## References

I used a few bits of code and documentation to write this, such as:

- https://www.ni.com/en/support/documentation/supplemental/06/the-ni-tdms-file-format.html
- https://www.ni.com/en/support/documentation/supplemental/07/tdms-file-format-internal-structure.html
- https://www.ni.com/docs/en-US/bundle/labview/page/tdm-data-model.html
- https://www.ni.com/en/support/documentation/supplemental/06/introduction-to-labview-tdm-streaming-vis.html
- https://www.ni.com/docs/en-US/bundle/labwindows-cvi/page/cvi/libref/cvitdmslibrary.htm
- https://github.com/ni/nidaqmx-python
- https://github.com/ni/tdms-parser/
- https://github.com/adamreeve/npTDMS/
