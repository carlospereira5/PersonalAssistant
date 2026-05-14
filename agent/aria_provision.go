// Package agent — aria_provision.go gestiona el lifecycle de módulos en startup.
package agent

import (
	"context"
	"errors"
)

// provisionModules ejecuta el lifecycle de cada módulo registrado.
// Soporta Gen2 via type assertions: DataReader, DataWriter, backgroundModule.
func (a *Aria) provisionModules() {
	deps := a.buildDeps()

	for _, m := range a.modules {
		modLogger := a.logger.WithPrefix(m.Name())

		for _, stmt := range m.Schema() {
			if _, err := a.db.Exec(stmt); err != nil {
				modLogger.Warn("schema migration skipped", "module", m.Name(), "err", err)
			}
		}

		depsWithLogger := deps
		depsWithLogger.Logger = modLogger
		if err := m.Init(depsWithLogger); err != nil {
			modLogger.Error("Init fallido", "err", err)
			continue
		}

		var toolCount int

		if r, ok := m.(DataReader); ok {
			a.registerReadTools(r)
			toolCount += len(r.ReadTools())
		}

		if w, ok := m.(DataWriter); ok {
			a.registerDataWriter(w)
			toolCount += len(w.WriteTools())
		}

		modLogger.Info("Módulo registrado", "tools", toolCount)
	}
}

// buildDeps construye el PortDeps base provisionado a cada módulo.
func (a *Aria) buildDeps() PortDeps {
	return PortDeps{
		DB:            a.db,
		Logger:        a.logger,
		LLM:           a.llm,
		BackgroundLLM: a.bgLLM,
		Messenger:     a.messenger,
		Config:        a.repos.Config,
		Tasks:         a.repos.Tasks,
		Reminders:     a.repos.Reminders,
		Scheduler:     a.repos.Scheduler,
	}
}

func (a *Aria) registerReadTools(r DataReader) {
	for _, tool := range r.ReadTools() {
		toolName := tool.Name
		reader := r
		a.registry.Register(tool, func(ctx context.Context, args map[string]any) (map[string]any, error) {
			return reader.Read(ctx, toolName, args)
		})
	}
}

func (a *Aria) registerDataWriter(w DataWriter) {
	for _, tool := range w.WriteTools() {
		toolName := tool.Name
		writer := w
		a.registry.Register(tool, func(ctx context.Context, args map[string]any) (map[string]any, error) {
			return writer.Write(ctx, toolName, args)
		})
	}
}

// Start arranca los módulos con background loops.
func (a *Aria) Start(ctx context.Context) {
	for _, m := range a.modules {
		bg, ok := m.(backgroundModule)
		if !ok {
			continue
		}
		name := m.Name()
		go func(bg backgroundModule) {
			if err := bg.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
				a.logger.Error("background module stopped", "module", name, "err", err)
			}
		}(bg)
	}
}

// SetMessenger permite inyectar o reemplazar el Messenger en runtime.
func (a *Aria) SetMessenger(m Messenger) {
	a.messenger = m
	for _, mod := range a.modules {
		if ms, ok := mod.(interface{ SetMessenger(Messenger) }); ok {
			ms.SetMessenger(m)
		}
	}
}
