module github.com/septagon-oss/pk-shared

go 1.26

retract v0.0.0 // broken: contained local replace directives

// v0.3.0 was withdrawn: its history was rewritten to correct commit
// attribution, so the tag no longer resolves to the content the module proxy
// recorded. v0.4.0 is the same code with clean history.
retract v0.3.0
