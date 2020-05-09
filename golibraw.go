package golibraw

// #cgo LDFLAGS: -lraw
// #include "libraw/libraw.h"
import "C"
import (
	"bytes"
	"fmt"
	"image"
	"log"
	"os"
	"time"
	"unsafe"

	"github.com/lmittmann/ppm"
)

// ImgMetadata contiene alcuni dati relativi all'immagine letti da libraw
type ImgMetadata struct {
	ScattoTimestamp int64
	ScattoDataOra   string
}

type rawImg struct {
	Height   int
	Width    int
	Bits     uint
	DataSize int
	Data     []byte
}

func (r rawImg) fullBytes() []byte {
	header := fmt.Sprintf("P6\n%d %d\n%d\n", r.Width, r.Height, (1<<r.Bits)-1)
	return append([]byte(header), r.Data...)
}

func handleError(msg string, err int) {
	if err != 0 {
		fmt.Printf("ERROR libraw  %v\n", C.libraw_strerror(C.int(err)))
	}
}

func lrInit() *C.libraw_data_t {
	librawProcessor := C.libraw_init(0)
	return librawProcessor
}

// ExportEmbeddedJPEG immagine JPEG salvata in file RAW
func ExportEmbeddedJPEG(inputPath string, inputfile os.FileInfo, exportPath string) (string, error) {

	outfile := exportPath + "/" + inputfile.Name() + "_embedded.jpg"
	infile := inputPath + "/" + inputfile.Name()

	if _, err := os.Stat(outfile); os.IsNotExist(err) {
		librawProcessor := lrInit()
		C.libraw_open_file(librawProcessor, C.CString(infile))

		ret := C.libraw_unpack_thumb(librawProcessor)
		handleError("unpack thumb", int(ret))

		//ret = C.libraw_dcraw_process(iprc)
		//handleError("process", int(ret))
		//iprc.params.output_tiff = 1
		//outfile := exportPath + "/" + inputfile.Name() + ".tiff"

		//fmt.Printf("exporting %s  ->  %s \n", inputfile.Name(), outfile)
		ret = C.libraw_dcraw_thumb_writer(librawProcessor, C.CString(outfile))

		handleError("save thumb", int(ret))

		C.libraw_recycle(librawProcessor)
		lrClose(librawProcessor)
	}
	return outfile, nil
}

// Raw2Image creates a Image from raw file
func Raw2Image(infile string) (image.Image, ImgMetadata, error) {
	t0 := time.Now()

	librawProcessor := lrInit()

	C.libraw_open_file(librawProcessor, C.CString(infile))

	ret := C.libraw_unpack(librawProcessor)
	handleError("unpack", int(ret))

	ret = C.libraw_dcraw_process(librawProcessor)
	handleError("dcraw processing", int(ret))

	var makeImageErr C.int

	//typedef struct
	//{
	//  enum LibRaw_image_formats type;
	//  ushort height, width, colors, bits;
	//  unsigned int data_size;
	//  unsigned char data[1];
	//} libraw_processed_image_t;
	//
	myImage := C.libraw_dcraw_make_mem_image(librawProcessor, &makeImageErr)
	handleError("dcraw processing", int(makeImageErr))

	dataBytes := make([]uint8, int(myImage.data_size))

	// in C sta usando un flexible array ... non so come accedervi in golang, così però sembra funzionare
	start := unsafe.Pointer(&myImage.data)
	size := unsafe.Sizeof(uint8(0))
	for i := 0; i < int(myImage.data_size); i++ {
		item := *(*uint8)(unsafe.Pointer(uintptr(start) + size*uintptr(i)))
		dataBytes[i] = item
		// fmt.Printf("%d => %d \n", i, item)
	}

	rawImage := rawImg{
		Height:   int(myImage.height),
		Width:    int(myImage.width),
		DataSize: int(myImage.data_size),
		Bits:     uint(myImage.bits),
		Data:     dataBytes,
	}
	/*
		outfilename := fmt.Sprintf(".rawtool/%s.ppm", inputfile.Name())
		f, err := os.Create(outfilename)
		if err != nil {
			fmt.Println(err)
			return nil, fmt.Errorf("errore in creazione file out")
		}

		n2, err := f.Write(rawImage.fullBytes())
		if err != nil {
			fmt.Println(err)
			f.Close()
			return nil, fmt.Errorf("errore in scrittura file out")
		}
		fmt.Println(n2, "bytes written successfully")
		err = f.Close()
	*/

	//iparam := C.libraw_get_iparams(librawProcessor)
	//log.Printf(" iparam  = %v", iparam)
	//lensinfo := C.libraw_get_lensinfo(librawProcessor)
	//log.Printf(" lensinfo  = %v", lensinfo)
	other := C.libraw_get_imgother(librawProcessor)
	//log.Printf(" OTHER = %v", other.timestamp)

	// data di scatto (timestamp)
	timestamp := int64(other.timestamp)
	dataScatto := time.Unix(timestamp, 0)

	C.libraw_dcraw_clear_mem(myImage)
	C.libraw_recycle(librawProcessor)

	log.Printf("    raw decoding required %v", time.Since(t0))
	fullbytes := rawImage.fullBytes()
	result, err := ppm.Decode(bytes.NewReader(fullbytes))

	return result,
		ImgMetadata{ScattoTimestamp: timestamp,
			ScattoDataOra: dataScatto.Format("2006-01-02T15:04:05")}, err
	//outfile := "./" + inputfile.Name() + ".ppm"
	//fmt.Printf("exporting %s  ->  %s \n", inputfile.Name(), outfile)
	//ret = C.libraw_dcraw_ppm_tiff_writer(iprc, C.CString(outfile))

	//handleError("save ppm", int(ret))

	//}

	// return nil, nil
}

func Export(inputPath string, inputfile os.FileInfo, exportPath string) error {

	// FIXME controllare che file input esiste

	// lanciare errore se file input non esiste

	outfile := exportPath + "/" + inputfile.Name() + ".ppm"
	infile := inputPath + "/" + inputfile.Name()

	if _, err := os.Stat(outfile); os.IsNotExist(err) {
		librawProcessor := lrInit()
		C.libraw_open_file(librawProcessor, C.CString(infile))

		ret := C.libraw_unpack(librawProcessor)
		handleError("unpack", int(ret))

		ret = C.libraw_dcraw_process(librawProcessor)

		handleError("dcraw processing", int(ret))
		//iprc.params.output_tiff = 1
		//outfile := exportPath + "/" + inputfile.Name() + ".tiff"

		fmt.Printf("exporting %s  ->  %s \n", inputfile.Name(), outfile)
		ret = C.libraw_dcraw_ppm_tiff_writer(librawProcessor, C.CString(outfile))

		handleError("save ppm", int(ret))

		C.libraw_recycle(librawProcessor)
		lrClose(librawProcessor)
	}
	return nil
}

func lrClose(iprc *C.libraw_data_t) {
	C.libraw_close(iprc)
}
