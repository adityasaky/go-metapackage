# go-metapackage

This is a small tool written as part of a larger experiment. Warning: it is NOT intended for any professional use.

A "metapackage" in this instance is a package that doesn't actually do anything. When supplied a dependency, the tool builds a "main" package that uses every public API provided by the dependency. The main package generated can be compiled but will almost certainly panic when executed.