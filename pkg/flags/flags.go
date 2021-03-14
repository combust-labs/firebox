package flags

import (
	"github.com/go-openapi/runtime/flagext"
	"github.com/jessevdk/go-flags"
	flag "github.com/spf13/pflag"
)

type FlagSet struct {
	*flag.FlagSet
}

func ExtendFlagSet(flagSet *flag.FlagSet) *FlagSet {
	return &FlagSet{flagSet}
}

func (f *FlagSet) ByteSizeVar(p *flagext.ByteSize, name string, value flagext.ByteSize, usage string) {
	f.VarP(newByteSizeValue(value, p), name, "", usage)
}

func newByteSizeValue(val flagext.ByteSize, p *flagext.ByteSize) *flagext.ByteSize {
	*p = val
	return p
}

func (f *FlagSet) FilenameVar(p *flags.Filename, name string, value flags.Filename, usage string) {
	f.VarP(newFilename(value, p), name, "", usage)
}

func newFilename(val flags.Filename, p *flags.Filename) *Filename {
	*p = val
	return &Filename{p}
}

type Filename struct {
	*flags.Filename
}

func (b Filename) String() string {
	return string(*b.Filename)
}

func (b *Filename) Set(value string) error {
	*b.Filename = flags.Filename(value)
	return nil
}

func (b *Filename) Type() string {
	return "filename"
}
