package main

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Shopify/go-lua"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"
)

type Function struct {
	tfprotov6.Function
	Impl func(args []*tfprotov6.DynamicValue) (*tfprotov6.DynamicValue, *tfprotov6.FunctionError)
}

type FunctionProvider struct {
	ProviderSchema   *tfprotov6.Schema
	StaticFunctions  map[string]*Function
	dynamicFunctions map[string]*Function
	Configure        func(*tfprotov6.DynamicValue) (map[string]*Function, []*tfprotov6.Diagnostic)
}

func (f *FunctionProvider) GetMetadata(context.Context, *tfprotov6.GetMetadataRequest) (*tfprotov6.GetMetadataResponse, error) {
	var functions []tfprotov6.FunctionMetadata
	for name := range f.StaticFunctions {
		functions = append(functions, tfprotov6.FunctionMetadata{Name: name})
	}

	return &tfprotov6.GetMetadataResponse{
		ServerCapabilities: &tfprotov6.ServerCapabilities{GetProviderSchemaOptional: true},
		Functions:          functions,
	}, nil
}
func (f *FunctionProvider) GetProviderSchema(context.Context, *tfprotov6.GetProviderSchemaRequest) (*tfprotov6.GetProviderSchemaResponse, error) {
	functions := make(map[string]*tfprotov6.Function)
	for name, fn := range f.StaticFunctions {
		functions[name] = &fn.Function
	}

	return &tfprotov6.GetProviderSchemaResponse{
		ServerCapabilities: &tfprotov6.ServerCapabilities{GetProviderSchemaOptional: true},
		Provider:           f.ProviderSchema,
		Functions:          functions,
	}, nil
}
func (f *FunctionProvider) ValidateProviderConfig(ctx context.Context, req *tfprotov6.ValidateProviderConfigRequest) (*tfprotov6.ValidateProviderConfigResponse, error) {
	// Passthrough
	return &tfprotov6.ValidateProviderConfigResponse{PreparedConfig: req.Config}, nil
}
func (f *FunctionProvider) ConfigureProvider(ctx context.Context, req *tfprotov6.ConfigureProviderRequest) (*tfprotov6.ConfigureProviderResponse, error) {
	funcs, diags := f.Configure(req.Config)
	f.dynamicFunctions = funcs
	return &tfprotov6.ConfigureProviderResponse{
		Diagnostics: diags,
	}, nil
}
func (f *FunctionProvider) StopProvider(context.Context, *tfprotov6.StopProviderRequest) (*tfprotov6.StopProviderResponse, error) {
	return &tfprotov6.StopProviderResponse{}, nil
}
func (f *FunctionProvider) ValidateResourceConfig(context.Context, *tfprotov6.ValidateResourceConfigRequest) (*tfprotov6.ValidateResourceConfigResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) UpgradeResourceState(context.Context, *tfprotov6.UpgradeResourceStateRequest) (*tfprotov6.UpgradeResourceStateResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ReadResource(context.Context, *tfprotov6.ReadResourceRequest) (*tfprotov6.ReadResourceResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) PlanResourceChange(context.Context, *tfprotov6.PlanResourceChangeRequest) (*tfprotov6.PlanResourceChangeResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ApplyResourceChange(context.Context, *tfprotov6.ApplyResourceChangeRequest) (*tfprotov6.ApplyResourceChangeResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ImportResourceState(context.Context, *tfprotov6.ImportResourceStateRequest) (*tfprotov6.ImportResourceStateResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ValidateDataResourceConfig(context.Context, *tfprotov6.ValidateDataResourceConfigRequest) (*tfprotov6.ValidateDataResourceConfigResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ReadDataSource(context.Context, *tfprotov6.ReadDataSourceRequest) (*tfprotov6.ReadDataSourceResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) CallFunction(ctx context.Context, req *tfprotov6.CallFunctionRequest) (*tfprotov6.CallFunctionResponse, error) {
	if fn, ok := f.StaticFunctions[req.Name]; ok {
		ret, err := fn.Impl(req.Arguments)
		return &tfprotov6.CallFunctionResponse{
			Result: ret,
			Error:  err,
		}, nil
	}
	if f.dynamicFunctions != nil {
		if fn, ok := f.dynamicFunctions[req.Name]; ok {
			ret, err := fn.Impl(req.Arguments)
			return &tfprotov6.CallFunctionResponse{
				Result: ret,
				Error:  err,
			}, nil
		}
	}
	return nil, errors.New("unknown function " + req.Name)
}
func (f *FunctionProvider) GetFunctions(context.Context, *tfprotov6.GetFunctionsRequest) (*tfprotov6.GetFunctionsResponse, error) {
	functions := make(map[string]*tfprotov6.Function)
	for name, fn := range f.StaticFunctions {
		functions[name] = &fn.Function
	}
	for name, fn := range f.dynamicFunctions {
		functions[name] = &fn.Function
	}

	return &tfprotov6.GetFunctionsResponse{
		Functions: functions,
	}, nil
}

func main() {
	err := tf6server.Serve("registry.opentofu.org/opentofu/lua", func() tfprotov6.ProviderServer {
		provider := &FunctionProvider{
			ProviderSchema: &tfprotov6.Schema{
				Block: &tfprotov6.SchemaBlock{
					Attributes: []*tfprotov6.SchemaAttribute{
						&tfprotov6.SchemaAttribute{
							Name:     "lua",
							Type:     tftypes.String,
							Required: true,
						},
					},
				},
			},
			Configure: func(config *tfprotov6.DynamicValue) (map[string]*Function, []*tfprotov6.Diagnostic) {
				res, err := config.Unmarshal(tftypes.Map{ElementType: tftypes.String})
				if err != nil {
					return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "Invalid configure payload",
						Detail:   err.Error(),
					}}
				}
				cfg := make(map[string]tftypes.Value)
				err = res.As(&cfg)
				if err != nil {
					return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "Invalid configure payload",
						Detail:   err.Error(),
					}}
				}

				codeVal := cfg["lua"]
				var code string
				err = codeVal.As(&code)
				if err != nil {
					return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "Invalid configure payload",
						Detail:   err.Error(),
					}}
				}

				functions := make(map[string]*Function)

				// This is terrible, but I'm new to lua CAPI
				regex := regexp.MustCompile(`function (.*)\(`)
				funcs := regex.FindAllStringSubmatch(code, -1)
				for _, bfn := range funcs {
					fn := strings.TrimSpace(bfn[1])
					functions[fn] = &Function{
						tfprotov6.Function{
							VariadicParameter: &tfprotov6.FunctionParameter{
								AllowNullValue: true,
								Name:           "args",
								Type:           tftypes.DynamicPseudoType,
							},
							Return: &tfprotov6.FunctionReturn{
								Type: tftypes.DynamicPseudoType,
							},
						},
						func(args []*tfprotov6.DynamicValue) (*tfprotov6.DynamicValue, *tfprotov6.FunctionError) {
							l := lua.NewState()
							lua.OpenLibraries(l)

							// Load lua code
							if err := lua.DoString(l, code); err != nil {
								return nil, &tfprotov6.FunctionError{Text: err.Error()}
							}

							// Setup function call
							l.Global(fn)

							// Check valid function loaded
							if !l.IsFunction(-1) {
								return nil, &tfprotov6.FunctionError{Text: `missing or invalid "return <function>" at end of input`}
							}

							// Load arguments
							for _, arg := range args {
								err := ProtoToLua(arg, l)
								if err != nil {
									return nil, &tfprotov6.FunctionError{Text: err.Error()}
								}
							}

							// Call function, expecting one return value
							l.Call(len(args), 1)

							// Retrieve result
							val, err := LuaToProto(l)
							if err != nil {
								return nil, &tfprotov6.FunctionError{Text: err.Error()}
							}
							return val, nil
						},
					}
				}

				return functions, nil
			},
			StaticFunctions: map[string]*Function{
				"exec": &Function{
					tfprotov6.Function{
						Parameters: []*tfprotov6.FunctionParameter{&tfprotov6.FunctionParameter{
							Name: "code",
							Type: tftypes.String,
						}},
						VariadicParameter: &tfprotov6.FunctionParameter{
							AllowNullValue: true,
							Name:           "args",
							Type:           tftypes.DynamicPseudoType,
						},
						Return: &tfprotov6.FunctionReturn{
							Type: tftypes.DynamicPseudoType,
						},
					},
					func(args []*tfprotov6.DynamicValue) (*tfprotov6.DynamicValue, *tfprotov6.FunctionError) {
						codeVal, err := args[0].Unmarshal(tftypes.String)
						if err != nil {
							return nil, &tfprotov6.FunctionError{Text: err.Error()}
						}

						var code string
						err = codeVal.As(&code)
						if err != nil {
							return nil, &tfprotov6.FunctionError{Text: err.Error()}
						}
						args = args[1:]

						l := lua.NewState()
						lua.OpenLibraries(l)

						// Load lua code
						if err := lua.DoString(l, code); err != nil {
							return nil, &tfprotov6.FunctionError{Text: err.Error()}
						}

						// Check valid function loaded
						if !l.IsFunction(-1) {
							return nil, &tfprotov6.FunctionError{Text: `missing or invalid "return <function>" at end of input`}
						}

						// Load arguments
						for _, arg := range args {
							err := ProtoToLua(arg, l)
							if err != nil {
								return nil, &tfprotov6.FunctionError{Text: err.Error()}
							}
						}

						// Call function, expecting one return value
						l.Call(len(args), 1)

						// Retrieve result
						val, err := LuaToProto(l)
						if err != nil {
							return nil, &tfprotov6.FunctionError{Text: err.Error()}
						}
						return val, nil
					},
				},
			},
		}
		return provider
	})
	if err != nil {
		panic(err)
	}
}

func ProtoToLua(arg *tfprotov6.DynamicValue, l *lua.State) error {
	argCty, err := ProtoToCty(arg)
	if err != nil {
		return err
	}

	return CtyToLua(argCty, l)
}

func LuaToProto(l *lua.State) (*tfprotov6.DynamicValue, error) {
	argCty, err := LuaToCty(l)
	if err != nil {
		return nil, err
	}
	return CtyToProto(argCty)
}

func ProtoToCty(arg *tfprotov6.DynamicValue) (cty.Value, error) {
	// Decode using cty directly as it supports DynamicPseudoType
	// This is inspired by github.com/apparentlymart/go-tf-func-provider
	if len(arg.MsgPack) != 0 {
		return msgpack.Unmarshal(arg.MsgPack, cty.DynamicPseudoType)
	}
	if len(arg.JSON) != 0 {
		return json.Unmarshal(arg.JSON, cty.DynamicPseudoType)
	}
	panic("unknown encoding")
}

func CtyToProto(ctyVal cty.Value) (*tfprotov6.DynamicValue, error) {
	result, err := msgpack.Marshal(ctyVal, cty.DynamicPseudoType)
	if err != nil {
		return nil, err
	}
	return &tfprotov6.DynamicValue{
		MsgPack: result,
	}, nil
}

func CtyToLua(arg cty.Value, l *lua.State) error {
	switch t := arg.Type(); t {
	case cty.Number:
		var v float64
		err := gocty.FromCtyValue(arg, &v)
		if err != nil {
			return err
		}
		l.PushNumber(v)
		return nil
	case cty.String:
		var v string
		err := gocty.FromCtyValue(arg, &v)
		if err != nil {
			return err
		}
		l.PushString(v)
		return nil
	case cty.Bool:
		var v bool
		err := gocty.FromCtyValue(arg, &v)
		if err != nil {
			return err
		}
		l.PushBoolean(v)
		return nil
	default:
		if t.IsObjectType() || t.IsMapType() {
			l.NewTable()
			for k, v := range arg.AsValueMap() {
				l.PushString(k)
				err := CtyToLua(v, l)
				if err != nil {
					return err
				}
				l.SetTable(-3)
			}
			return nil
		}
		if t.IsListType() || t.IsSetType() || t.IsTupleType() {
			l.NewTable()
			for k, v := range arg.AsValueSlice() {
				l.PushInteger(k)
				err := CtyToLua(v, l)
				if err != nil {
					return err
				}
				l.SetTable(-3)
			}
			return nil
		}
		return fmt.Errorf("unsupported parameter type %#v", arg.Type())
	}
}

func LuaToCty(l *lua.State) (cty.Value, error) {
	if l.IsNone(-1) {
		return cty.NilVal, fmt.Errorf("none value should not be returned")
	}

	switch t := l.TypeOf(-1); t {
	case lua.TypeNil:
		return cty.NilVal, nil
	case lua.TypeBoolean:
		return cty.BoolVal(l.ToBoolean(-1)), nil
	case lua.TypeNumber:
		number, _ := l.ToNumber(-1)
		return cty.NumberFloatVal(number), nil
	case lua.TypeString:
		str, _ := l.ToString(-1)
		return cty.StringVal(str), nil
	case lua.TypeTable:
		// https://stackoverflow.com/a/6142700
		mv := make(map[string]cty.Value)

		// Space for key
		l.PushNil()

		// Push value
		for l.Next(-2) {
			// Copy key to top of stack
			l.PushValue(-2)

			// Decode key (also modifies)
			key, ok := l.ToString(-1)
			if !ok {
				return cty.NilVal, fmt.Errorf("bad table index")
			}

			l.Pop(1)

			// Decode Value (also modifies)
			val, err := LuaToCty(l)
			if err != nil {
				return cty.NilVal, err
			}
			mv[key] = val

			l.Pop(1)
		}

		av := make([]cty.Value, len(mv))

		off := 1
		// Hack in an off-by-one offset
		if _, ok := mv["0"]; ok {
			off = 0
		}

		// This is inefficient, but it works
		for i := off; i < len(av)+off; i++ {
			if v, ok := mv[strconv.Itoa(i)]; ok {
				av[i] = v
			} else {
				// Not a coherent list
				return cty.ObjectVal(mv), nil
			}
		}
		return cty.ListVal(av), nil
	default:
		return cty.NilVal, fmt.Errorf("unhanded return type %s!", t)
	}
}
