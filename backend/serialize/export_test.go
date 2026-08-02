package serialize

// Test-only hook. Lets the external test package (serialize_test), which can import the
// products fixture, diff the production encoder against the retained tree oracle.
var MarshalTree = marshalTree
