package metapackage

import (
	"fmt"
	"go/token"
	"go/types"
	"unicode"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type FunctionStructure struct {
	Name          string
	Receiver      *types.Var
	Parameters    *types.Tuple
	Returns       *types.Tuple
	PkgImportPath string
}

func (f *FunctionStructure) ParentTypeName() (string, error) {
	if f.Receiver == nil {
		return "", fmt.Errorf("function has no receiver")
	}

	// FIXME: this can't use getTypeName just yet.
	var typeName string
	switch typ := f.Receiver.Type().(type) {
	case *types.Pointer:
		typeName = typ.Elem().(*types.Named).Obj().Name()
	case *types.Named:
		typeName = typ.Obj().Name()
	}
	return typeName, nil
}

func (f *FunctionStructure) IsReceiverPointer() bool {
	if f.Receiver == nil {
		return false
	}

	switch f.Receiver.Type().(type) {
	case *types.Pointer:
		return true
	default:
		return false
	}
}

func (f *FunctionStructure) IsParentTypePrivate() bool {
	if f.Receiver == nil {
		return false
	}

	typeName, _ := f.ParentTypeName()
	return unicode.IsLower([]rune(typeName)[0])
}

func FindAllFunctions(target string) ([]FunctionStructure, error) {
	allFunctions := []FunctionStructure{}
	allPackages, err := LoadPackages([]string{target}, false)
	if err != nil {
		return nil, err
	}

	prog, _ := ssautil.AllPackages(allPackages, 0)

	for _, pkg := range prog.AllPackages() {
		if pkg.Pkg.Path() == target {
			for _, member := range pkg.Members {
				if member.Token() == token.FUNC {
					function := member.(*ssa.Function)
					if unicode.IsLower([]rune(function.Name())[0]) {
						// All exported identifiers must be capitalized
						// Therefore, lower => private
						// ssa.Func does not have a boolean flag for some reason
						continue
					}
					newFunc := FunctionStructure{
						Name:          function.Name(),
						PkgImportPath: function.Pkg.Pkg.Path(),
						Receiver:      function.Signature.Recv(),
						Parameters:    function.Signature.Params(),
						Returns:       function.Signature.Results(),
					}
					allFunctions = append(allFunctions, newFunc)
				} else if member.Token() == token.TYPE {
					typ := member.(*ssa.Type)
					methodSet := prog.MethodSets.MethodSet(types.NewPointer(typ.Type()))
					for i := 0; i < methodSet.Len(); i++ {
						method := methodSet.At(i)
						function, ok := method.Obj().(*types.Func)
						if !ok {
							continue
						}
						if !function.Exported() {
							continue
						}
						signature := function.Type().(*types.Signature)
						newFunc := FunctionStructure{
							Name:          function.Name(),
							PkgImportPath: function.Pkg().Path(),
							Receiver:      signature.Recv(),
							Parameters:    signature.Params(),
							Returns:       signature.Results(),
						}
						allFunctions = append(allFunctions, newFunc)
					}
				}
			}
		}
	}

	return allFunctions, nil
}

func LoadPackages(targets []string, tests bool) ([]*packages.Package, error) {
	packageModes := packages.NeedName |
		packages.NeedFiles |
		packages.NeedCompiledGoFiles |
		packages.NeedImports |
		packages.NeedDeps |
		packages.NeedExportsFile |
		packages.NeedTypes |
		packages.NeedSyntax |
		packages.NeedTypesInfo |
		packages.NeedTypesSizes |
		packages.NeedModule

	pkgConfig := &packages.Config{
		Mode:  packageModes,
		Dir:   "",
		Tests: tests,
	}

	allPackages, err := packages.Load(pkgConfig, targets...)
	if err != nil {
		return nil, err
	}
	return allPackages, nil
}
