package ascmhl

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"path/filepath"
	"sort"

	"github.com/Avalanche-io/c4/id"
	"github.com/cespare/xxhash/v2"
	"github.com/spf13/afero"
	"github.com/zeebo/xxh3"
)

type FileHash struct {
	Hash map[string]string
	Size int64
	Time string
}

// GetMapHash finds all of the hashes for a folder's contents, it returns maps of the hashes of the content and their structure
func getMapHash(root string, reqHash []string) (map[string]FileHash, map[string]map[string]string, error) {

	_, exist := afero.ReadDir(AppFS, root)
	contentHash := make(map[string]FileHash)
	structureHash := make(map[string]map[string]string)

	root = filepath.Clean(root)

	if exist == nil {
		//afero.FullBaseFsPath(AppFS,root)
		//trueRoot, _ := filepath.Abs(root)

		// Extract the file strucuture
		names, contents, err := findFiles(root)
		if err != nil {
			return contentHash, structureHash, err
		}
		// Make maps and add the hash values

		for i, cont := range contents {
			// Create an array of the contents and structure of each folder
			folderContent := make([]map[string]string, len(cont))
			folderStructure := make([]map[string]string, len(cont))

			for j, c := range cont { // of cont calculate the struct value and content value
				// Make the path for each file in a folder per loop
				path := names[i] + "/" + c.Name
				var coHas FileHash
				var stHas map[string]string
				if c.Dir {
					// If a folder then we can extract the value as these have already been calculated
					// In a previous iteration
					d, _ := AppFS.Open(path)
					dirStat, _ := d.Stat()
					// Utilise this to get the time of the folder and include in the mhl
					coHas = contentHash[path]
					// Update folder time
					coHas.Time = dirStat.ModTime().Format(xmlTFormat)
					stHas = fileStructureHash(structureHash[path], c.Name)
				} else {
					// B, err := os.ReadFile(path)
					toHash, err := AppFS.Open(path)
					if err == nil {
						var b []byte
						coHas, b = getFileHash(toHash, reqHash)
						stHas = getStructureHash(b, reqHash, c.Name)
					} else {
						return nil, nil, err
					}
				} // Assign the generated map
				contentHash[path] = coHas
				folderContent[j] = coHas.Hash
				folderStructure[j] = stHas
			}

			// Calculate the content hash of the folder and add it to the map
			// Using an array of the hashes of the contents
			var content FileHash
			content.Hash = conHash(folderContent)
			contentHash[names[i]] = content
			// Repeat for structure
			strHash := conHash(folderStructure)
			structureHash[names[i]] = strHash
		}

	} else {
		toHash, exist := AppFS.Open(root)
		if exist == nil {
			// Process f as a file
			// Just include the basic hashlist etc
			f, _ := getFileHash(toHash, reqHash)
			contentHash[root] = f
		} else {
			return nil, nil, exist
		}
	}

	return contentHash, structureHash, nil
}

func getFileHash(file afero.File, want []string) (FileHash, []byte) {
	hashInf, _ := file.Stat()
	var f FileHash
	f.Time = hashInf.ModTime().Format(xmlTFormat)
	b, _ := io.ReadAll(file)
	f.Hash = contentHasher(b, want)
	f.Size = hashInf.Size()

	return f, b
}

// ContnentHash generates the content hash for a file
func contentHasher(b []byte, types []string) map[string]string {
	return hashGen(b, types)
}

// StructureHash returns a map of the code of the contents of the files added onto the name of the file
// the encoded again in the same code
func getStructureHash(b []byte, types []string, fname string) map[string]string {
	// Loop through
	// Get your hash
	cont := contentHasher(b, types)
	structure := make(map[string]string)

	for key, val := range cont {
		if key == "C4" {
			code, _ := id.Parse(val)
			ncode := append([]byte(fname), code.Digest()...)
			structure = addHash(ncode, key, structure)
		} else {
			code, _ := hex.DecodeString(val)
			ncode := append([]byte(fname), code...)
			structure = addHash(ncode, key, structure)
		}
	}

	return structure
}

// FileStrucutreHash takes the list of codes for a file and calulates the structure for it
// This is only used for the finding the strucuture of a folder with folders in it
func fileStructureHash(codes map[string]string, fname string) map[string]string {

	structure := make(map[string]string)
	// Manually check to see if things have been generated by types

	for key, val := range codes {
		if key == "C4" {
			code, _ := id.Parse(val)
			ncode := append([]byte(fname), code.Digest()...)
			structure = addHash(ncode, key, structure)
		} else {
			code, _ := hex.DecodeString(val)
			ncode := append([]byte(fname), code...)
			structure = addHash(ncode, key, structure)
		}
	}

	return structure
}

func hashGen(file []byte, types []string) map[string]string {
	h := make(map[string]string, 0)
	for _, s := range types {
		h = addHash(file, s, h)
	}

	return h
}

func addHash(b []byte, hash string, m map[string]string) map[string]string {

	switch hash {
	case "C4":
		id3 := id.Identify(bytes.NewReader(b))
		m["C4"] = id3.String()
	case "Md5":
		md := md5.Sum(b)
		m["Md5"] = hex.EncodeToString(md[:])
	case "Sha1":
		s1 := sha1.Sum(b)
		m["Sha1"] = hex.EncodeToString(s1[:])
	case "Sha256":
		s256 := sha256.Sum256(b)
		m["Sha256"] = hex.EncodeToString(s256[:])
	case "Xxh128":
		v := xxh3.Hash128(b)
		vb := v.Bytes()
		m["Xxh128"] = hex.EncodeToString(vb[:])
	case "Xxh3":
		v := xxh3.Hash(b)
		var mid8 [8]byte
		binary.BigEndian.PutUint64(mid8[:], v)
		m["Xxh3"] = hex.EncodeToString(mid8[:])
	case "Xxh64":
		v := xxhash.Sum64(b)
		var mid8 [8]byte
		binary.BigEndian.PutUint64(mid8[:], v)
		m["Xxh64"] = hex.EncodeToString(mid8[:])
	}

	return m
	// Add the extra features here like time etc? or put the time in for all of them
}

// ConHash takes an array of hashes
// , it sorts them in order, appends them and generates the relative hash
func conHash(h []map[string]string) map[string]string {
	hashArray := make(map[string][]string)
	// Convert a map of arrays to the
	for _, j := range h {
		for key, val := range j {

			hashArray[key] = append(hashArray[key], val)
		}
	}
	resultmap := make(map[string]string)
	for htype, hash := range hashArray {
		// Fmt.Println(len(hash) == len(h), len(hash), len(h))
		if len(hash) == len(h) {
			// Carry on all is good in the hood
			// Fmt.Println(hash)
			sort.Strings(hash)
			// Fmt.Println(hash)
			var sortedByte []byte
			for _, val := range hash {
				if htype == "C4" {
					code, _ := id.Parse(val)
					sortedByte = append(sortedByte, code.Digest()...)
				} else {
					b, _ := hex.DecodeString(val)
					sortedByte = append(sortedByte, b...)
				}
			}
			// Fmt.Println(sortedByte)
			resultmap = addHash(sortedByte, htype, resultmap)
		}
	}

	return resultmap
}
