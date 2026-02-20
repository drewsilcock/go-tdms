# go-tdms

[![CI](https://github.com/drewsilcock/go-tdms/actions/workflows/ci.yaml/badge.svg)](https://github.com/drewsilcock/go-tdms/actions/workflows/ci.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/drewsilcock/go-tdms.svg)](https://pkg.go.dev/github.com/drewsilcock/go-tdms)

This is a pure Go no-dependency* file parser for the Technical Data Management Streaming (TDMS) format used by National Instruments (NI) software such as LabVIEW.

*Technically, there's a test dependency, but I'm pretty sure that doesn't get included when you install the package.

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
		for value, err := range channel.ReadFloat64() {
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(value)
		}

		// Iterate through batches of values.
		for batch, err := range channel.ReadFloat64Batch() {
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(batch)
		}

		// Batch size is configurable (both for individual value streamer and
		// batch streamer)
		for batch, err := range channel.ReadFloat64Batch(tdms.BatchSize(1024)) {
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(batch)
		}

		// Read all values into a single slice
		values, err := channel.ReadFloat64All() {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(values)
	}
}

// Access property value string using the `As[Type]()` methods.
author, err := file.Properties.GetString("Author")
if err != nil {
	log.Fatal(err)
}
fmt.Println("Why, this TDMS file was written by none other than ", author)
```

### Batching

There are three method variants provided, depending on the output you want:

- `ch.Read()`, `ch.ReadFloat64()`, etc. – returns an iterator that gives individual values. Note that this still internally uses batching as an optimisation, it just unpacks it for your convenience.
- `ch.ReadBatch()`, `ch.ReadFloat64Batch()`, etc. – return an iterators that gives you the batches of values, where each batch corresponds to the batches used internally for optimisation.
- `ch.ReadAll()`, `ch.ReadFloat64All()` ,etc. – return all values as a single slice. This still uses batching internally for data parsing and scaling, but puts all the output data into a single slice. This is useful if you know you're dealing with files that fit comfortably in memory and don't need streaming.

It's possible to modify the batch size used with the `tdms.BatchSize(size)` option. Note that the batch size is the number of values, not the number of bytes, which means for data types which are particularly large, you may want to be considerate of keeping the batch size not too big. For instance, using a batch size of 1024 is eminently reasonable for float64 values but if they are strings, each string could potentially be 100 bytes, meaning the batch size is now 100 MB.

### Type safety

We provide both strongly typed functions and functions that return `any` when reading data.

If you call `ch.ReadFloat64()` on a channel where the data is not float64, the method will panic.

If you want to know the output type for a given channel given specified options, run `ch.OutputType(myOption, myOtherOption)`.

If you want to do your own type switch to handle all the different variants, use the method without any type name in its name, e.g. `ch.Read()`.

Remember: scaling can change the data type (e.g. linear scaling converts integers to floating point numbers). See the scaling section below for more details.

### DAQmx

DAQmx data scalers are supported, although they have not been battle tested (real TDMS files have many edge cases and inconsistencies which LabVIEW is happy with even though they're against the "spec").

Using `WithScaling(false)` on a DAQmx channel has no effect as the scalers are required to understand the data.

When reading DAQmx data, you use exactly the same read methods as non-DAQmx data, with the single difference that you should specify which DAQmx scaler you want to read the data for. A single read method call will read the data for one scaler only. By default, `tdms` looks for scaler with scale index 0. This is configurable using `ForDAQmxScaler(scaleIndex)`.

For instance, reading float64 data from a channel for scale index 1:

```go
for value, err := range ch.ReadFloat64(ForDAQmxScaler(1)) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(value)
}
```

If the scaler with the specified scale index does not exist, this will yield an error.

### Scaling

TDMS supports scaling. LabVIEW includes many scalers such as [strain scalers](https://www.ni.com/en/shop/data-acquisition/sensor-fundamentals/measuring-strain-with-strain-gages.html) for sensors that measure deformation of materials, linear scalers for simple ax+b mathematical scalings, and more.

This library supports all of the scalers mentioned on the NI documentation, in addition to the AddScaling and SubtractScaling which are not mentioned.

Our implemention of scaling is based on the fantastic [npTDMS](https://github.com/adamreeve/npTDMS/) library, where the author has reverse engineered the scaling functionality, with the minor addition of the reciprocal scaler which npTDMS doesn't currently support.

By default, the read methods will apply any scaling that is found in the metadata. If you don't want this, you can specify `WithScaling(false)` as an option to your read function.

**Bear in mind** that applying scaling can change the output data type (e.g. linear scaler will convert raw data types that are int32 to float64). If you are checking the output type, do so using the `ch.OutputType(options...)` method and pass your scaling option into that argument. (This will also handle cases where the data is DAQmx data, where an internal conversion to the actual data type is needed based on the input DAQmx scale index.)

### Waveform

The channels contain y value data. If you want to also get the x value data (usually time values, but not always), you can use `channel.Waveform()`. This will read the [standard waveform properties used by LabVIEW](https://www.ni.com/docs/en-US/bundle/labview-api-ref/page/functions/tdms-set-properties.html) to determine the x-axis values for your channel.

```go
waveform, err := ch.Waveform()
if err != nil {
    log.Fatal(err)
}

// Say I want to know what the x-axis value is for index 347:
idx := 347
xaxisValue := waveform.Value(idx)
if err != nil {
    log.Fatal(err)
}

// If I know I'm dealing with time-series data, I can creator a time waveform
// which gives me time values:
timeWaveform, err := waveform.AsWaveform()
if err != nil {
    log.Fatal(err)
}

// I can get the time as a time.Time for a specific index:
timeValue := timeWaveform.Time(idx)

// You can also iterate through waveform values:
for ts := range timeWaveform.Times() {
	fmt.Println(ts)
}

// Or you can pull all time values from the waveform:
allTimes := timeWaveform.AllTimes()
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
| Data scaling                                | ☑️     |
| DAQmx data and scalers                      | ☑️ *   |
| Fixed point numerics                        | □ **   |

\* DAQmx functionality is working, however it has not been battle tested.

\** See note below on fixed point numerics.

### Future work

#### Benchmarking

We could do with adding benchmarks for all the different core pieces of functionality, mainly reading file metadata and reading file data.

It'd be interesting to explore performance with very large data files.

#### Fixed point numerics

The official documentation does not provide any detail on what format the fixed point numerics are stored on disk with, and I cannot find any examples of TDMS files with fixed point numerics on the internet, so until I can find more information this is going to remain unimplemented.

## References

I used a few bits of code and documentation to write this, such as:

- https://www.ni.com/en/support/documentation/supplemental/06/the-ni-tdms-file-format.html
- https://www.ni.com/en/support/documentation/supplemental/07/tdms-file-format-internal-structure.html
- https://www.ni.com/docs/en-US/bundle/labview/page/tdm-data-model.html
- https://www.ni.com/en/support/documentation/supplemental/06/introduction-to-labview-tdm-streaming-vis.html
- https://www.ni.com/docs/en-US/bundle/labwindows-cvi/page/cvi/libref/cvitdmslibrary.htm
- https://www.ni.com/en/support/documentation/supplemental/18/ni-daqmx-custom-scales-and-usage-explained.html
- https://www.ni.com/en/shop/data-acquisition/sensor-fundamentals/measuring-strain-with-strain-gages.html#toc1
- https://github.com/ni/nidaqmx-python
- https://github.com/ni/tdms-parser/
- https://github.com/adamreeve/npTDMS/
- https://github.com/adamreeve/rstdms
