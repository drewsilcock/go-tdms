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
//			for value, err := range channel.ReadDataAsFloat64() {
//				if err != nil {
//					log.Fatal(err)
//				}
//				fmt.Println(value)
//			}
//		}
//	}
package tdms
