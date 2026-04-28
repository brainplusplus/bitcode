package qjs_runtime

import (
	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
	"github.com/fastschema/qjs"
)

func registerHostFunctions(ctx *qjs.Context, bc *bridge.Context) {
	ctx.SetFunc("__bc_env", func(this *qjs.This) (*qjs.Value, error) {
		key := this.Args()[0].String()
		val, err := bc.Env(key)
		if err != nil {
			return nil, err
		}
		return ctx.NewStringHandle(val), nil
	})

	ctx.SetFunc("__bc_config", func(this *qjs.This) (*qjs.Value, error) {
		key := this.Args()[0].String()
		val := bc.Config(key)
		return qjs.ToJsValue(ctx, val)
	})

	ctx.SetFunc("__bc_log", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		level := args[0].String()
		msg := args[1].String()
		bc.Log(level, msg)
		return ctx.NewUndefined(), nil
	})

	ctx.SetFunc("__bc_t", func(this *qjs.This) (*qjs.Value, error) {
		key := this.Args()[0].String()
		return ctx.NewStringHandle(bc.T(key)), nil
	})

	ctx.SetFunc("__bc_emit", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		event := args[0].String()
		data, _ := qjs.ToGoValue[map[string]any](args[1])
		err := bc.Emit(event, data)
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})

	ctx.SetFunc("__bc_call", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		process := args[0].String()
		input, _ := qjs.ToGoValue[map[string]any](args[1])
		result, err := bc.Call(process, input)
		if err != nil {
			return nil, err
		}
		return qjs.ToJsValue(ctx, result)
	})

	ctx.SetFunc("__bc_db_query", func(this *qjs.This) (*qjs.Value, error) {
		sql := this.Args()[0].String()
		result, err := bc.DB().Query(sql)
		if err != nil {
			return nil, err
		}
		return qjs.ToJsValue(ctx, result)
	})

	ctx.SetFunc("__bc_db_execute", func(this *qjs.This) (*qjs.Value, error) {
		sql := this.Args()[0].String()
		result, err := bc.DB().Execute(sql)
		if err != nil {
			return nil, err
		}
		return qjs.ToJsValue(ctx, result)
	})

	ctx.SetFunc("__bc_http_request", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		method := args[0].String()
		url := args[1].String()
		var opts *bridge.HTTPOptions
		if len(args) > 2 && !args[2].IsUndefined() {
			raw, _ := qjs.ToGoValue[map[string]any](args[2])
			opts = embedded.ParseHTTPOpts(raw)
		}
		var result *bridge.HTTPResponse
		var err error
		switch method {
		case "GET":
			result, err = bc.HTTP().Get(url, opts)
		case "POST":
			result, err = bc.HTTP().Post(url, opts)
		case "PUT":
			result, err = bc.HTTP().Put(url, opts)
		case "PATCH":
			result, err = bc.HTTP().Patch(url, opts)
		case "DELETE":
			result, err = bc.HTTP().Delete(url, opts)
		}
		if err != nil {
			return nil, err
		}
		return qjs.ToJsValue(ctx, result)
	})

	ctx.SetFunc("__bc_cache_get", func(this *qjs.This) (*qjs.Value, error) {
		key := this.Args()[0].String()
		val, err := bc.Cache().Get(key)
		if err != nil {
			return nil, err
		}
		return qjs.ToJsValue(ctx, val)
	})

	ctx.SetFunc("__bc_cache_set", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		key := args[0].String()
		val, _ := qjs.ToGoValue[any](args[1])
		var opts *bridge.CacheSetOptions
		if len(args) > 2 && !args[2].IsUndefined() {
			raw, _ := qjs.ToGoValue[map[string]any](args[2])
			opts = embedded.ParseCacheOpts(raw)
		}
		err := bc.Cache().Set(key, val, opts)
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})

	ctx.SetFunc("__bc_cache_del", func(this *qjs.This) (*qjs.Value, error) {
		err := bc.Cache().Del(this.Args()[0].String())
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})

	registerFSFunctions(ctx, bc)
	registerModelFunctions(ctx, bc)
	registerMiscFunctions(ctx, bc)
}

func registerFSFunctions(ctx *qjs.Context, bc *bridge.Context) {
	ctx.SetFunc("__bc_fs_read", func(this *qjs.This) (*qjs.Value, error) {
		content, err := bc.FS().Read(this.Args()[0].String())
		if err != nil {
			return nil, err
		}
		return ctx.NewStringHandle(content), nil
	})
	ctx.SetFunc("__bc_fs_write", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		err := bc.FS().Write(args[0].String(), args[1].String())
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})
	ctx.SetFunc("__bc_fs_exists", func(this *qjs.This) (*qjs.Value, error) {
		exists, err := bc.FS().Exists(this.Args()[0].String())
		if err != nil {
			return nil, err
		}
		return ctx.NewBool(exists), nil
	})
	ctx.SetFunc("__bc_fs_list", func(this *qjs.This) (*qjs.Value, error) {
		files, err := bc.FS().List(this.Args()[0].String())
		if err != nil {
			return nil, err
		}
		return qjs.ToJsValue(ctx, files)
	})
	ctx.SetFunc("__bc_fs_mkdir", func(this *qjs.This) (*qjs.Value, error) {
		err := bc.FS().Mkdir(this.Args()[0].String())
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})
	ctx.SetFunc("__bc_fs_remove", func(this *qjs.This) (*qjs.Value, error) {
		err := bc.FS().Remove(this.Args()[0].String())
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})
}

func registerModelFunctions(ctx *qjs.Context, bc *bridge.Context) {
	ctx.SetFunc("__bc_model_search", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		name := args[0].String()
		raw, _ := qjs.ToGoValue[map[string]any](args[1])
		result, err := bc.Model(name).Search(embedded.ParseSearchOpts(raw))
		if err != nil {
			return nil, err
		}
		return qjs.ToJsValue(ctx, result)
	})
	ctx.SetFunc("__bc_model_get", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		result, err := bc.Model(args[0].String()).Get(args[1].String())
		if err != nil {
			return nil, err
		}
		return qjs.ToJsValue(ctx, result)
	})
	ctx.SetFunc("__bc_model_create", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		data, _ := qjs.ToGoValue[map[string]any](args[1])
		result, err := bc.Model(args[0].String()).Create(data)
		if err != nil {
			return nil, err
		}
		return qjs.ToJsValue(ctx, result)
	})
	ctx.SetFunc("__bc_model_write", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		data, _ := qjs.ToGoValue[map[string]any](args[2])
		err := bc.Model(args[0].String()).Write(args[1].String(), data)
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})
	ctx.SetFunc("__bc_model_delete", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		err := bc.Model(args[0].String()).Delete(args[1].String())
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})
	ctx.SetFunc("__bc_model_count", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		raw, _ := qjs.ToGoValue[map[string]any](args[1])
		count, err := bc.Model(args[0].String()).Count(embedded.ParseSearchOpts(raw))
		if err != nil {
			return nil, err
		}
		return ctx.NewInt64(count), nil
	})
}

func registerMiscFunctions(ctx *qjs.Context, bc *bridge.Context) {
	ctx.SetFunc("__bc_exec", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		cmd := args[0].String()
		cmdArgs, _ := qjs.ToGoValue[[]string](args[1])
		var opts *bridge.ExecOptions
		if len(args) > 2 && !args[2].IsUndefined() {
			raw, _ := qjs.ToGoValue[map[string]any](args[2])
			opts = embedded.ParseExecOpts(raw)
		}
		result, err := bc.Exec(cmd, cmdArgs, opts)
		if err != nil {
			return nil, err
		}
		return qjs.ToJsValue(ctx, result)
	})

	ctx.SetFunc("__bc_email_send", func(this *qjs.This) (*qjs.Value, error) {
		raw, _ := qjs.ToGoValue[map[string]any](this.Args()[0])
		err := bc.Email().Send(embedded.ParseEmailOpts(raw))
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})

	ctx.SetFunc("__bc_notify_send", func(this *qjs.This) (*qjs.Value, error) {
		raw, _ := qjs.ToGoValue[map[string]any](this.Args()[0])
		err := bc.Notify().Send(embedded.ParseNotifyOpts(raw))
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})

	ctx.SetFunc("__bc_notify_broadcast", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		channel := args[0].String()
		data, _ := qjs.ToGoValue[map[string]any](args[1])
		err := bc.Notify().Broadcast(channel, data)
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})

	ctx.SetFunc("__bc_crypto_hash", func(this *qjs.This) (*qjs.Value, error) {
		hash, err := bc.Crypto().Hash(this.Args()[0].String())
		if err != nil {
			return nil, err
		}
		return ctx.NewStringHandle(hash), nil
	})

	ctx.SetFunc("__bc_crypto_verify", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		ok, err := bc.Crypto().Verify(args[0].String(), args[1].String())
		if err != nil {
			return nil, err
		}
		return ctx.NewBool(ok), nil
	})

	ctx.SetFunc("__bc_audit_log", func(this *qjs.This) (*qjs.Value, error) {
		raw, _ := qjs.ToGoValue[map[string]any](this.Args()[0])
		err := bc.Audit().Log(embedded.ParseAuditOpts(raw))
		if err != nil {
			return nil, err
		}
		return ctx.NewUndefined(), nil
	})

	sessionVal, _ := qjs.ToJsValue(ctx, bc.Session())
	ctx.Global().SetPropertyStr("__bc_session", sessionVal)
}
