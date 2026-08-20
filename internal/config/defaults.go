package config

// Default holds every default value of the format, and is the only place they
// appear. Load decodes the file into this value rather than into a zero one:
// an absent key keeps its default, an explicit key overwrites it even when it
// carries zero. That distinction matters on privacy keys, where 0 means "no
// trimming at all" and absence means 3000 m.
