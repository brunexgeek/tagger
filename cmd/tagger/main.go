package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/brunexkgeek/tagger/internal/server"
)

type FileTags struct {
	Hash string `json:"h,omitempty"`
	Tags []int  `json:"t,omitempty"`
}

type Database struct {
	Version    [3]int               `json:"v,omitempty"`
	Entries    map[string]*FileTags `json:"e,omitempty"`
	Tags       map[int]string       `json:"t,omitempty"`
	TagsByName map[string]int       `json:"-"`
	LastTag    int                  `json:"l,omitempty"`
}

func loadDatabase(filename string) (Database, error) {
	var db Database
	db.Entries = make(map[string]*FileTags)
	db.Tags = make(map[int]string)
	db.TagsByName = make(map[string]int)
	db.Version = [3]int{1, 0, 0}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return db, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return db, err
	}

	err = json.Unmarshal(data, &db)
	if err != nil {
		return db, err
	}

	if db.Version != [3]int{1, 0, 0} {
		return db, fmt.Errorf("invalid database version")
	}

	for tag, name := range db.Tags {
		db.TagsByName[name] = tag
	}

	return db, nil
}

func saveDatabase(filename string, db Database) error {
	data, err := json.Marshal(db)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func addFileTags(db *Database, fileName string, tags []string) {
	// find the entry
	entry, ok := db.Entries[fileName]
	if !ok {
		entry = &FileTags{Hash: "", Tags: make([]int, 0)}
		db.Entries[fileName] = entry
	}

	for _, tag := range tags {
		// create unknown tags
		tid, ok := db.TagsByName[tag]
		if !ok {
			db.LastTag += 1
			tid = db.LastTag
			db.TagsByName[tag] = tid
			db.Tags[tid] = tag
		}
		// append the new tag
		// TODO: check for existing tags
		entry.Tags = append(entry.Tags, tid)
	}
}

func searchByTag(db Database, tag string) []string {
	tid, ok := db.TagsByName[tag]
	if !ok {
		return []string{}
	}

	var result []string
	for fileName, current := range db.Entries {
		for _, t := range current.Tags {
			if t == tid {
				result = append(result, fileName)
				break
			}
		}
	}
	return result
}

func help() {
	fmt.Println("Usage: tagger --add <path> <tag1> [tag2] ... [tagN]")
	fmt.Println("       tagger --tag <tag> <path1> [path2] ... [pathN]")
	fmt.Println("       tagger --find <tag>")
	fmt.Println("       tagger --status <path>")
	fmt.Println("       tagger --list")
}

func expandPath(root string, path string) (string, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(resolved, root) {
		return "", fmt.Errorf("path is not relative to '%s'", root)
	}
	_, err = os.Stat(resolved)
	if err != nil {
		return "", err
	}
	resolved, _ = strings.CutPrefix(resolved, root)
	return resolved, nil
}

func main() {
	if len(os.Args) < 2 {
		help()
		return
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Println("unable to get working directory")
		return
	}
	root, err = filepath.Abs(root)
	if err != nil {
		fmt.Println("unable to get absolute working directory")
		return
	}
	dbFile := filepath.Join(root, ".tagger")
	operation := os.Args[1]

	db, err := loadDatabase(dbFile)
	if err != nil {
		fmt.Println("Error loading database:", err)
		return
	}

	if operation == "--server" {
		server.Serve("/home/bruno/.cache/thumbnails/normal")
	} else if operation == "--find" {
		tag := os.Args[2]
		results := searchByTag(db, tag)
		if len(results) == 0 {
			fmt.Println("No files found with tag:", tag)
		} else {
			fmt.Printf("Files with tag '%s':\n", tag)
			for _, path := range results {
				fmt.Println("  ", filepath.Join(root, path))
			}
		}
	} else if operation == "--list" {
		fmt.Println("Existing tags:")
		for tag := range db.TagsByName {
			fmt.Println("  ", tag)
		}
	} else if operation == "--status" {
		filename, err := expandPath(root, os.Args[2])
		if err != nil {
			fmt.Println(err)
			return
		}

		entry := db.Entries[filename]
		if entry == nil {
			fmt.Println("File not tagged")
		} else {
			fmt.Printf("Existing tags for '%s':\n", filename)
			for _, tag := range entry.Tags {
				tagName, ok := db.Tags[tag]
				if !ok {
					fmt.Println("  ", tag)
				} else {
					fmt.Println("  ", tagName)
				}
			}
		}
	} else if operation == "--add" {
		if len(os.Args) < 4 {
			help()
			return
		}

		fileName, err := expandPath(root, os.Args[2])
		if err != nil {
			fmt.Println(err)
			return
		}
		tags := os.Args[3:]

		addFileTags(&db, fileName, tags)

		err = saveDatabase(dbFile, db)
		if err != nil {
			fmt.Println("Error saving database:", err)
			return
		}

		fmt.Println("File tags added successfully")
	} else if operation == "--tag" {
		if len(os.Args) < 4 {
			help()
			return
		}

		tags := []string{os.Args[2]}
		files := os.Args[3:]
		for _, filename := range files {
			filename, err := expandPath(root, filename)
			if err != nil {
				fmt.Println(err)
			} else {
				addFileTags(&db, filename, tags)
				fmt.Printf("Added tag '%s' to '%s'\n", tags[0], filename)
			}
		}

		err = saveDatabase(dbFile, db)
		if err != nil {
			fmt.Println("Error saving database:", err)
			return
		}
	} else {
		help()
	}
}
