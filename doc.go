// Package tdms provides a pure Go parser for the Technical Data Management
// Streaming (TDMS) file format used by National Instruments (NI) software such
// as LabVIEW.
//
// Open a file with [Open] or create a [File] from an [io.ReadSeeker] with [New].
// Access groups and channels via the [File.Groups] map, then read channel data
// using the typed streaming, batch, or read-all methods on [Channel].
//
//	file, err := tdms.Open("data.tdms")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer file.Close()
//
//	for _, group := range file.Groups {
//		for _, channel := range group.Channels {
//			// Iterate through individual values (uses batching internally).
//			for value, err := range channel.ReadDataAsFloat64() {
//				if err != nil {
//					log.Fatal(err)
//				}
//				fmt.Println(value)
//			}
//
//			// Iterate through batches of values
//			for batch, err := range channel.ReadDataAsFloat64Batch() {
//				if err != nil {
//					log.Fatal(err)
//				}
//				fmt.Println(batch)
//			}
//
//			// Batch size is configurable (both for individual value streamer and
//			// batch streamer)
//			for batch, err := range channel.ReadDataAsFloat64Batch(tdms.BatchSize(1024)) {
//				if err != nil {
//					log.Fatal(err)
//				}
//				fmt.Println(batch)
//			}
//
//			// Read all values into a single slice
//			values, err := channel.ReadDataAsFloat64All() {
//			if err != nil {
//				log.Fatal(err)
//			}
//			fmt.Println(values)
//		}
//	}
//
// Files, groups, and channels can all have properties. To get a type-safe
// property value, use the `As[Type]()` methods, e.g. [Property.AsFloat64],
// [Property.AsUint32], [Property.AsString], etc.
//
//	authorProp := file.Properties["Author"]
//
//	// Don't confuse `String()` (Stringer interface implementation) with
//	// `AsString()`, which returns the value as a string.
//	author, err := authorProp.AsString()
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Timestamps are stored as tdms.Timestamp, which is more precise than time.Time.
// You can easily convert from one to the other using [Timestamp.AsTime].
// Property values can be retrieved as their TDMS timestamp using
// [Property.AsTimestamp] or automatically converted to [time.Time] using
// [Property.AsTime].
//
//	createdAtProp, err := file.Properties["CreatedAt]
//	createdAt, err := createdAtProp.AsTimestamp()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	createdAtTime := createdAt.AsTime()
//	fmt.Printf("File was created at %s", createdAtTime)
//
// TDMS supports 128-bit extended precision floating point numbers. To do
// arithmetic with these, you can either convert them to float64 (losing precision)
// or convert them to big.Float, maintaining full precision at the cost of making
// it a bit more fiddly to work with. This applies equally to properties and data.
//
//	calibrationFactorProp, err := channel.Properties["CalibrationFactor"]
//	calibrationFactor, err := calibrationFactorProp.AsFloat128()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	calibrationFactorBigFloat := calibrationFactor.AsBigFloat()
//	fmt.Printf("Calibration factor is %s", calibrationFactorBigFloat)
//
// You can also simply get the value as [any] and perform your own switch on the
// type. This is an exhaustive list of all the possible types that [tdms]
// currently supports:
//
//	prop := file.Properties["analysisResults"]
//	switch v := prop.Value.(type) {
//	case int8:
//		fmt.Printf("Analysis results are 8-bit signed integer: %v", v)
//	case int16:
//		fmt.Printf("Analysis results are 16-bit signed integer: %v", v)
//	case int32:
//		fmt.Printf("Analysis results are 32-bit signed integer: %v", v)
//	case int64:
//		fmt.Printf("Analysis results are 64-bit signed integer: %v", v)
//	case uint8:
//		fmt.Printf("Analysis results are 8-bit unsigned integer: %v", v)
//	case uint16:
//		fmt.Printf("Analysis results are 16-bit unsigned integer: %v", v)
//	case uint32:
//		fmt.Printf("Analysis results are 32-bit unsigned integer: %v", v)
//	case uint64:
//		fmt.Printf("Analysis results are 64-bit unsigned integer: %v", v)
//	case float32:
//		fmt.Printf("Analysis results are 32-bit floating point: %v", v)
//	case float64:
//		fmt.Printf("Analysis results are 64-bit floating point: %v", v)
//	case tdms.Float128:
//		fmt.Printf("Analysis results are 128-bit floating point: %v", v)
//	case string:
//		fmt.Printf("Analysis results are string: %v", v)
//	case bool:
//		fmt.Printf("Analysis results are boolean: %v", v)
//	case tdms.Timestamp:
//		fmt.Printf("Analysis results are timestamp: %v", v)
//	case complex64:
//		fmt.Printf("Analysis results are 64-bit complex floating point: %v", v)
//	case complex128:
//		fmt.Printf("Analysis results are 128-bit complex floating point: %v", v)
//	default:
//		fmt.Printf("Analysis results are of unknown type: %T", v)
//	}
//
// When opening a [File] from a filename with [File.Open], the file is
// determined to be an index file (i.e. containing all metadata and no raw data)
// if the filename ends with `.tdms_index`. Otherwise, it's supposed to be a
// standard TDMS file with data in.
//
// As well as opening files with [File.Open], you can also open files with
// [File.New], passing any type that implements the `io.ReadSeeker` interface
// (i.e. you can read values and seek). When you do this, we can no longer infer
// the total file size and whether the file is an index file or not, so you need
// to pass those in. For example:
//
//	var tdmsFileBytes []byte
//	tdmsReader := bytes.NewReader(tdmsFileBytes)
//	tdmsSize := len(tdmsFileBytes)
//	isIndex := false
//
//	file, err := tdms.New(tdmsReader, isIndex, tdmsSize)
//	if err != nil {
//		log.Fatal(err)
//	}
package tdms
