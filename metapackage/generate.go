package metapackage

import (
	"fmt"
	"go/types"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/dave/jennifer/jen"
)

type Declare struct {
	Name    string
	Package string
	Type    types.Type
	Pointer bool
}

// TODO: is this global variable the best way?
var closures []FunctionStructure

func cleanUpFuncNames(name string) string {
	// TODO: find a better way
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, "(", "_")
	name = strings.ReplaceAll(name, ")", "_")
	name = strings.ReplaceAll(name, "#", "_")

	// Add replacements for other illegal characters

	return name
}

func generateVariableName(length int) string {
	alphabet := "abcdefghijklmnopqrstuvwxyz"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = alphabet[r.Intn(len(alphabet))]
	}
	return string(b)
}

// FIXME: getTypeName is doing too much, inconsistent with the use of jen to generate code elsewhere
func getTypeName(t types.Type) (string, error) {
	switch typ := t.(type) {
	case *types.Basic:
		return typ.Name(), nil
	case *types.Named:
		if typ.Obj().Pkg() == nil {
			return typ.Obj().Name(), nil
		}
		return typ.Obj().Pkg().Name() + "." + typ.Obj().Name(), nil
	case *types.Pointer:
		switch pointsTo := typ.Elem().(type) {
		case *types.Basic:
			return "*" + pointsTo.Name(), nil
		case *types.Named:
			return "*" + pointsTo.Obj().Pkg().Name() + "." + pointsTo.Obj().Name(), nil
		default:
			fmt.Printf("UNKNOWN POINTER TYPE NAME %T", pointsTo)
		}
	case *types.Slice:
		elemType, err := getTypeName(typ.Elem())
		if err != nil {
			return "", err
		}
		return "[]" + elemType, nil
	case *types.Array:
		elemType, err := getTypeName(typ.Elem())
		if err != nil {
			return "", err
		}
		return "[" + strconv.Itoa(int(typ.Len())) + "]" + elemType, nil
	case *types.Map:
		keyType, err := getTypeName(typ.Key())
		if err != nil {
			return "", err
		}
		valType, err := getTypeName(typ.Elem())
		if err != nil {
			return "", err
		}
		return "map[" + keyType + "]" + valType, nil
	case *types.Interface:
		return "interface{}", nil
	case *types.Signature:
		return "func" + typ.Params().String() + " " + typ.Results().String(), nil
	default:
		fmt.Printf("UNKNOWN REGULAR TYPE NAME %T", typ)
	}
	return "", fmt.Errorf("unhandled type")
}

func genBasic(declare *Declare) []*jen.Statement {
	var s []*jen.Statement
	if declare.Pointer {
		s = append(s, jen.Var().Id(declare.Name).Op("*").Id(declare.Type.(*types.Basic).Name()))
	} else {
		s = append(s, jen.Var().Id(declare.Name).Id(declare.Type.(*types.Basic).Name()))
	}
	return s
}

func genNamed(declare *Declare) []*jen.Statement {
	var s []*jen.Statement
	typ := declare.Type.(*types.Named)
	if typ.Obj().Pkg() != nil {
		if declare.Pointer {
			s = append(s, jen.Var().Id(declare.Name).Op("*").Id(typ.Obj().Pkg().Name()).Dot(typ.Obj().Name()))
		} else {
			s = append(s, jen.Var().Id(declare.Name).Id(typ.Obj().Pkg().Name()).Dot(typ.Obj().Name()))
		}
	} else {
		if declare.Pointer {
			s = append(s, jen.Var().Id(declare.Name).Op("*").Id(typ.Obj().Name()))
		} else {
			s = append(s, jen.Var().Id(declare.Name).Id(typ.Obj().Name()))
		}
	}
	return s
}

func genPointer(declare *Declare) []*jen.Statement {
	var s []*jen.Statement
	typ := declare.Type.(*types.Pointer)
	switch pointsTo := typ.Elem().(type) {
	case *types.Basic:
		s = genBasic(&Declare{Name: declare.Name, Package: declare.Package, Pointer: true, Type: pointsTo})
	case *types.Named:
		s = genNamed(&Declare{Name: declare.Name, Package: declare.Package, Pointer: true, Type: pointsTo})
	}
	return s
}

func genArray(declare *Declare) []*jen.Statement {
	var s []*jen.Statement
	typ := declare.Type.(*types.Array)
	// TODO: err handle
	arrayType, err := getTypeName(typ.Elem())
	if err != nil {
		fmt.Printf("Error: getTypeName can't handle a type")
	}
	// the type name includes a * if pointer
	s = append(s, jen.Var().Id(declare.Name).Index(jen.Lit(int(typ.Len()))).Id(arrayType))
	return s
}

func genSlice(declare *Declare) []*jen.Statement {
	var s []*jen.Statement
	typ := declare.Type.(*types.Slice)
	// TODO: err handle
	sliceType, err := getTypeName(typ.Elem())
	if err != nil {
		fmt.Printf("Error: getTypeName can't handle a type")
	}
	// the type name includes a * if pointer
	s = append(s, jen.Var().Id(declare.Name).Index().Id(sliceType))
	return s
}

func genMap(declare *Declare) []*jen.Statement {
	var s []*jen.Statement
	typ := declare.Type.(*types.Map)
	keyT := typ.Key()
	keyType, err := getTypeName(keyT)
	if err != nil {
		fmt.Printf("Error: getTypeName can't handle a type")
	}
	valT := typ.Elem()
	valType, err := getTypeName(valT)
	if err != nil {
		fmt.Printf("Error: getTypeName can't handle a type")
	}
	// the type name includes a * if pointer
	s = append(s, jen.Var().Id(declare.Name).Map(jen.Id(keyType)).Id(valType))
	return s
}

func genChan(declare *Declare) []*jen.Statement {
	var s []*jen.Statement
	elemT := declare.Type.(*types.Chan).Elem()
	elemType, _ := getTypeName(elemT)
	// the type name includes a * if pointer
	s = append(s, jen.Id(declare.Name).Op(":=").Make(jen.Chan().Id(elemType)))
	return s
}

func genInterface(declare *Declare) []*jen.Statement {
	var s []*jen.Statement
	s = append(s, jen.Var().Id(declare.Name).Interface())
	return s
}

func genStruct(declare *Declare) []*jen.Statement {
	var s []*jen.Statement
	s = append(s, jen.Var().Id(declare.Name).Struct())
	return s
}

func genDecl(declare *Declare) (string, []*jen.Statement) {
	if len(declare.Name) == 0 {
		declare.Name = generateVariableName(6)
	}
	var s []*jen.Statement
	switch typ := declare.Type.(type) {
	case *types.Basic:
		s = genBasic(declare)
	case *types.Named:
		s = genNamed(declare)
	case *types.Pointer:
		s = genPointer(declare)
	case *types.Array:
		s = genArray(declare)
	case *types.Slice:
		s = genSlice(declare)
	case *types.Map:
		s = genMap(declare)
	case *types.Chan:
		s = genChan(declare)
	case *types.Interface:
		s = genInterface(declare)
	case *types.Struct:
		s = genStruct(declare)
	case *types.Signature:
		// closure functions will be generated later
		declare.Name = generateVariableName(10)
		closures = append(closures, FunctionStructure{
			Name:          declare.Name,
			PkgImportPath: declare.Package,
			Parameters:    typ.Params(),
			Returns:       typ.Results(),
			Receiver:      typ.Recv(),
		})
	default:
		fmt.Printf("%T type for variable %s unsupported\n", typ, declare.Name)
	}
	return declare.Name, s
}

// Returns name of variable as it may have been generated
func generateVariableDeclaration(param *types.Var) (string, []*jen.Statement) {
	return genDecl(&Declare{
		Name:    param.Name(),
		Package: param.Pkg().Name(),
		Type:    param.Type(),
		Pointer: false,
	})

}

func GenerateSources(allFunctions []FunctionStructure) map[string][]*jen.Statement {
	sources := map[string][]*jen.Statement{}
	for _, fn := range allFunctions {
		var fnName string
		if strings.Contains(fn.Name, "#") {
			fnName = strings.Split(fn.Name, "#")[0]
		}
		if strings.Contains(fn.Name, "$") {
			fnName = strings.Split(fn.Name, "$")[0]
		}
		if fnName == "init" || fnName == "main" {
			continue
		}

		var name string
		parentType, err := fn.ParentTypeName()
		if err != nil {
			name = cleanUpFuncNames(fn.PkgImportPath + "_" + fn.Name)
		} else {
			name = cleanUpFuncNames(fn.PkgImportPath + "_" + parentType + "_" + fn.Name)
		}

		source := GenerateSource(fn)

		if len(source) > 0 {
			sources[name] = source
		}
	}

	return sources
}

func GenerateSource(function FunctionStructure) []*jen.Statement {
	stmts := []*jen.Statement{}
	params := function.Parameters
	funcparams := []jen.Code{}
	recv := function.Receiver

	if function.IsParentTypePrivate() {
		return stmts // len == 0
	}

	for i := 0; i < params.Len(); i++ {
		param := params.At(i)
		varName, decl := generateVariableDeclaration(param) // var name can change for closures
		stmts = append(stmts, decl...)
		funcparams = append(funcparams, jen.Id(varName))
	}

	if recv != nil {
		typeName, _ := function.ParentTypeName()

		ptr := function.IsReceiverPointer()

		var r *jen.Statement
		if ptr {
			r = jen.Var().Id(recv.Name()).Op("*").Id(recv.Pkg().Name()).Dot(typeName)
		} else {
			r = jen.Var().Id(recv.Name()).Id(recv.Pkg().Name()).Dot(typeName)
		}
		stmts = append(stmts, r)

		stmts = append(stmts, jen.Id(recv.Name()).Dot(function.Name).Call(funcparams...))
	} else {
		stmts = append(stmts, jen.Qual(function.PkgImportPath, function.Name).Call(funcparams...))
	}

	return stmts
}

func GenerateClosure(function FunctionStructure) ([]jen.Code, *jen.Statement, []jen.Code, error) {
	funcParams := []jen.Code{}
	for i := 0; i < function.Parameters.Len(); i++ {
		param := function.Parameters.At(i)
		typeName, err := getTypeName(param.Type())
		if err != nil {
			return nil, nil, nil, err
		}
		funcParams = append(funcParams, jen.Id(typeName))
	}

	source := []*jen.Statement{}
	returnParams := []jen.Code{}
	returnTypes := []string{}
	for i := 0; i < function.Returns.Len(); i++ {
		returnValue := function.Returns.At(i)
		varName, decl := generateVariableDeclaration(returnValue)
		source = append(source, decl...)
		typeName, err := getTypeName(returnValue.Type())
		if err != nil {
			return nil, nil, nil, err
		}
		returnParams = append(returnParams, jen.Id(varName))
		returnTypes = append(returnTypes, typeName)
	}
	source = append(source, jen.Return(returnParams...))

	sourceCode := []jen.Code{}
	for _, item := range source {
		sourceCode = append(sourceCode, item)
	}

	return funcParams, jen.Id(strings.Join(returnTypes, ",")), sourceCode, nil
}

func GeneratePackage(sources map[string][]*jen.Statement, path string) error {
	f := jen.NewFile("main")
	callsFromMain := []jen.Code{}
	for name, source := range sources {
		sourceCode := []jen.Code{}
		for _, item := range source {
			sourceCode = append(sourceCode, item)
		}
		f.Func().Id(name).Params().Block(sourceCode...)

		callsFromMain = append(callsFromMain, jen.Qual("", name).Call())
	}

	for _, c := range closures {
		funcParams, returns, source, err := GenerateClosure(c)
		if err != nil {
			return err
		}
		f.Func().Id(c.Name).Params(funcParams...).Parens(returns).Block(source...)
	}

	f.Func().Id("main").Params().Block(callsFromMain...)
	fmt.Printf("%#v", f)

	if err := f.Save(path); err != nil {
		return err
	}
	return nil
}
