package domain

type CollectionItemType string

const (
	CollectionItemRequest CollectionItemType = "request"
	CollectionItemFolder  CollectionItemType = "folder"
)

// CollectionItem is either a Request or a Folder; Type discriminates which.
type CollectionItem struct {
	Type    CollectionItemType
	Request *Request
	Folder  *Folder
}

type Folder struct {
	Name  string
	Items []CollectionItem
}

// Collection is a named, hierarchical set of requests.
// DefaultHeaders and Auth cascade to member requests unless overridden.
type Collection struct {
	SchemaVersion  string
	Name           string
	Description    string
	Items          []CollectionItem
	Variables      []Variable
	Auth           *Auth
	DefaultHeaders []Header
}
