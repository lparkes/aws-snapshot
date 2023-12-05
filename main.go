package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sync"
	//"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	//"github.com/aws/aws-sdk-go-v2/aws/awserr"
	//"github.com/aws/aws-sdk-go-v2/aws/session"
	"github.com/aws/aws-sdk-go-v2/service/ebs"
	"github.com/aws/aws-sdk-go-v2/service/ebs/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// Block contains the fields from ebs.PutSnapshotBlockInput that vary
// by block in a form that is easy to use rather than the vile AWS SDK
// style.
//
// The field ChecksumAlgorithm is also included because separating out
// knowledge of the name of the checksum algorithm from the code that
// calculate the checksum seems unwise.
type Block struct {
	// Constant across all blocks

	ChecksumAlgorithm string

	// Variable across all blocks

	BlockData  []byte
	BlockIndex int32
	Checksum   string
	Progress   int32 // The only optional field
}

const GiB = 1024 * 1024 * 1024

// Rename some of AWS' fucked up constant names.
const (
	SHA256 = types.ChecksumAlgorithmChecksumAlgorithmSha256
	LINEAR = types.ChecksumAggregationMethodChecksumAggregationLinear
)

var dFlag = flag.String("d", "NetBSD AMI", "Description for the snapshot being created")
var wFlag = flag.Int("w", 100, "Number of concurrent writers")

var debugFlag = flag.Bool("debug", false, "Enable debug logging")
var debug *log.Logger

func main() {
	flag.Parse()

	if *debugFlag {
		debug = log.New(os.Stderr, "debug ", log.LstdFlags)
	} else {
		debug = log.New(io.Discard, "", log.LstdFlags)
	}

	debug.Println("Debug logging enabled")

	// Configure all out channels
	blockNums := make(chan int32)
	unwrittenBlocks := make(chan *ebs.PutSnapshotBlockInput, 10)
	writtenBlocks := make(chan *ebs.PutSnapshotBlockInput)

	// Setup the local side of things.
	filename := flag.Arg(0)
	file, fileLength := OpenFile(filename)
	defer file.Close()

	// Setup the AWS side of things.
	ebsSvc := EbsSetup()
	snapshot := NewSnapshot(ebsSvc, fileLength)
	snapshotId := aws.ToString(snapshot.SnapshotId)
	blockSize := aws.ToInt32(snapshot.BlockSize)

	go EnumerateBlocks(blockNums, blockSize, fileLength)
	go BuildBlocks(blockNums, unwrittenBlocks, file, snapshot)
	go LaunchBlockWriters(ebsSvc, unwrittenBlocks, writtenBlocks, *wFlag)
	SnapshotFinisher(ebsSvc, writtenBlocks, snapshotId)

}

// OpenFile opens a file and returns the file pointer and the length
// of the file.
func OpenFile(filename string) (*os.File, int64) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	fi, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}

	return f, fi.Size()
}

func EbsSetup() *ebs.Client {
	cfg, _ := config.LoadDefaultConfig(context.TODO())
	return ebs.NewFromConfig(cfg, func(o *ebs.Options) {
		endpoint := os.Getenv("EBS_ENDPOINT")
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
}

func Ec2Setup() *ec2.Client {
	cfg, _ := config.LoadDefaultConfig(context.TODO())
	return ec2.NewFromConfig(cfg, func(o *ec2.Options) {
		endpoint := os.Getenv("EC2_ENDPOINT")
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
}

func NewSnapshot(c *ebs.Client, l int64) *ebs.StartSnapshotOutput {
	ssi := &ebs.StartSnapshotInput{
		ClientToken: aws.String(fmt.Sprintf("%d", rand.Int63())),
		Description: dFlag,
		VolumeSize:  aws.Int64((l + GiB - 1) / GiB),
	}

	sso, err := c.StartSnapshot(context.TODO(), ssi)
	if err != nil {
		log.Fatal(err)
	}

	return sso
}

func DeleteSnapshot(SnapshotId *string) {
	c := Ec2Setup() // Only needed to delete snapshots
	dsi := &ec2.DeleteSnapshotInput{
		SnapshotId: SnapshotId,
	}
	_, err := c.DeleteSnapshot(context.TODO(), dsi)
	if err != nil {
		log.Fatal(err)
	}
}

// EnumerateBlocks enumerates all the blocks in a file.
// The output channel is closed once all the blocks have been enumerated.
func EnumerateBlocks(blockNums chan<- int32, blockSize int32, fileSize int64) {
	blockCount := int32((fileSize + int64(blockSize) - 1) / int64(blockSize))

	debug.Println("Creating", blockCount, "block numbers")

	for i := int32(0); i < blockCount; i++ {
		blockNums <- i
	}

	close(blockNums)
	debug.Println("Finished creating", blockCount, "block numbers")
}

// BuildBlocks reads numbered blocks from f.
// The blocks channel is closed by BuildBlocks before it returns.
func BuildBlocks(blockNums <-chan int32, blocks chan<- *ebs.PutSnapshotBlockInput,
	f io.ReaderAt, sso *ebs.StartSnapshotOutput) {

	blockSize := aws.ToInt32(sso.BlockSize)

	for i := range blockNums {

		debug.Println("Reading block", i)

		blockData := make([]byte, int(blockSize))
		_, err := f.ReadAt(blockData, int64(blockSize)*int64(i))
		if err != nil {
			log.Fatal(err)
		}

		sum := sha256.Sum256(blockData)

		b := &ebs.PutSnapshotBlockInput{
			BlockData:         bytes.NewReader(blockData),
			BlockIndex:        aws.Int32(i),
			Checksum:          aws.String(base64.StdEncoding.EncodeToString(sum[:])),
			ChecksumAlgorithm: SHA256,
			DataLength:        sso.BlockSize,
			SnapshotId:        sso.SnapshotId,
		}

		blocks <- b
	}

	close(blocks)
	debug.Println("Finished reading blocks")
}

// WriteSnapshotBlocks writes snapshot blocks to the snapshot and then
// sends the blocks onwards for checksum aggregation.
func WriteSnapshotBlocks(svc *ebs.Client, blocksIn <-chan *ebs.PutSnapshotBlockInput, blocksDone chan<- *ebs.PutSnapshotBlockInput, wg *sync.WaitGroup) {
	ctx := context.TODO()

	for b := range blocksIn {

		debug.Println("Writing block", aws.ToInt32(b.BlockIndex))

		_, err := svc.PutSnapshotBlock(ctx, b)
		if err != nil {
			log.Fatal(err)
		}
		blocksDone <- b
	}

	wg.Done()
}

func LaunchBlockWriters(svc *ebs.Client, blocksIn <-chan *ebs.PutSnapshotBlockInput, blocksDone chan<- *ebs.PutSnapshotBlockInput, nWriters int) {
	wg := &sync.WaitGroup{}

	debug.Println("Launching", nWriters, "writers")

	wg.Add(nWriters)
	for i := 0; i < nWriters; i++ {
		go WriteSnapshotBlocks(svc, blocksIn, blocksDone, wg)
	}

	debug.Println("Waiting for writers to finish")
	wg.Wait()
	debug.Println("All writing done")
	close(blocksDone)
}

// SnapshotFinisher summarises the list of blocks that have been
// written to the snapshot and uses that summary to complete the
// snapshot.
func SnapshotFinisher(svc *ebs.Client, blocks <-chan *ebs.PutSnapshotBlockInput, snapshotId string) {
	var blockCount int32

	// The checksum aggregations need to be in order, so keep
	// track of where we are up to and make a place to store any
	// checksums we aren't ready for.
	nextCsumBlock := int32(0) // Same type as blockCount
	pendingCsums := make(map[int32]string)
	aggregateCsum := sha256.New()

	for b := range blocks {
		blockCount += 1
		blockIndex := aws.ToInt32(b.BlockIndex)
		pendingCsums[blockIndex] = aws.ToString(b.Checksum)
		for {
			sum, found := pendingCsums[nextCsumBlock]
			if !found {
				break
			}
			delete(pendingCsums, nextCsumBlock)
			aggregateCsum.Write([]byte(sum))
			nextCsumBlock += 1
		}
	}

	csi := ebs.CompleteSnapshotInput{
		SnapshotId:         aws.String(snapshotId),
		ChangedBlocksCount: aws.Int32(blockCount),
		//Checksum:                  AwsBase64(aggregateCsum.Sum(nil)),
		ChecksumAggregationMethod: LINEAR,
		ChecksumAlgorithm:         SHA256,
	}

	ctx := context.TODO()
	_, err := svc.CompleteSnapshot(ctx, &csi)
	if err != nil {
		log.Fatal(err)
	}
	// The output status is inevitably "pending", which is never
	// interesting enough to note.
}

func AwsBase64(data []byte) *string {
	return aws.String(base64.StdEncoding.EncodeToString(data))
}
