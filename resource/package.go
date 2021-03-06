// Copyright (c) 2015-2017 Marin Atanasov Nikolov <dnaeon@gmail.com>
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
//
//  1. Redistributions of source code must retain the above copyright
//     notice, this list of conditions and the following disclaimer
//     in this position and unchanged.
//  2. Redistributions in binary form must reproduce the above copyright
//     notice, this list of conditions and the following disclaimer in the
//     documentation and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE AUTHOR(S) ``AS IS'' AND ANY EXPRESS OR
// IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES
// OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED.
// IN NO EVENT SHALL THE AUTHOR(S) BE LIABLE FOR ANY DIRECT, INDIRECT,
// INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT
// NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF
// THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// +build !windows

package resource

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/dnaeon/gru/utils"
)

// ErrNoPackageProviderFound is returned when no suitable provider is found
var ErrNoPackageProviderFound = errors.New("No suitable package provider found")

// BasePackage is the base resource type for package management
// It's purpose is to be embedded into other package resource providers.
type BasePackage struct {
	Base

	// Name of the package to manage. Defaults to the resource name.
	Package string `luar:"-"`

	// Version of the package.
	Version string `luar:"version"`

	// Package manager to use
	manager string `luar:"-"`

	// Arguments to use when quering a package
	queryArgs []string `luar:"-"`

	// Arguments to use when installing a package
	installArgs []string `luar:"-"`

	// Arguments to use when deinstalling a package
	deinstallArgs []string `luar:"-"`

	// Arguments to use when updating a package
	updateArgs []string `luar:"-"`
}

// Evaluate evaluates the state of the package
func (bp *BasePackage) Evaluate() (State, error) {
	s := State{
		Current: "unknown",
		Want:    bp.State,
	}

	_, err := exec.LookPath(bp.manager)
	if err != nil {
		return s, err
	}

	bp.queryArgs = append(bp.queryArgs, bp.Package)
	cmd := exec.Command(bp.manager, bp.queryArgs...)
	err = cmd.Run()

	if err != nil {
		s.Current = "deinstalled"
	} else {
		s.Current = "installed"
	}

	return s, nil
}

// Create installs the package
func (bp *BasePackage) Create() error {
	Logf("%s installing package\n", bp.ID())

	bp.installArgs = append(bp.installArgs, bp.Package)
	cmd := exec.Command(bp.manager, bp.installArgs...)
	out, err := cmd.CombinedOutput()

	for _, line := range strings.Split(string(out), "\n") {
		Logf("%s %s\n", bp.ID(), line)
	}

	return err
}

// Delete deletes the package
func (bp *BasePackage) Delete() error {
	Logf("%s removing package\n", bp.ID())

	bp.deinstallArgs = append(bp.deinstallArgs, bp.Package)
	cmd := exec.Command(bp.manager, bp.deinstallArgs...)
	out, err := cmd.CombinedOutput()

	for _, line := range strings.Split(string(out), "\n") {
		Logf("%s %s\n", bp.ID(), line)
	}

	return err
}

// Update updates the package
func (bp *BasePackage) Update() error {
	Logf("%s updating package\n", bp.ID())

	bp.updateArgs = append(bp.updateArgs, bp.Package)
	cmd := exec.Command(bp.manager, bp.updateArgs...)
	out, err := cmd.CombinedOutput()

	for _, line := range strings.Split(string(out), "\n") {
		Logf("%s %s\n", bp.ID(), line)
	}

	return err
}

// NewPackage creates a new resource for managing packages.
// This provider tries to determine the most appropriate
// package provider for you, so it is more like a meta-provider.
//
// Example:
//   pkg = resource.package.new("tmux")
//   pkg.state = "installed"
func NewPackage(name string) (Resource, error) {
	// Releases files used by the various GNU/Linux distros
	releases := map[string]Provider{
		"/etc/arch-release":   NewPacman,
		"/etc/centos-release": NewYum,
		"/etc/redhat-release": NewYum,
		"/usr/local/sbin/pkg": NewPkgNG,
	}

	// Do our best to determine the proper provider
	for release, provider := range releases {
		dst := utils.NewFileUtil(release)
		if dst.Exists() {
			return provider(name)
		}
	}

	return nil, ErrNoPackageProviderFound
}

// Pacman type represents the resource for package management on
// Arch Linux systems.
//
// Example:
//   pkg = resource.pacman.new("tmux")
//   pkg.state = "installed"
type Pacman struct {
	BasePackage
}

// NewPacman creates a new resource for managing packages
// using the pacman package manager on an Arch Linux system
func NewPacman(name string) (Resource, error) {
	p := &Pacman{
		BasePackage: BasePackage{
			Base: Base{
				Name:              name,
				Type:              "package",
				State:             "installed",
				Require:           make([]string, 0),
				PresentStatesList: []string{"present", "installed"},
				AbsentStatesList:  []string{"absent", "deinstalled"},
				Concurrent:        false,
				Subscribe:         make(TriggerMap),
			},
			Package:       name,
			Version:       "",
			manager:       "/usr/bin/pacman",
			queryArgs:     []string{"--query"},
			installArgs:   []string{"--sync", "--noconfirm"},
			deinstallArgs: []string{"--remove", "--noconfirm"},
			updateArgs:    []string{"--sync", "--noconfirm"},
		},
	}

	return p, nil
}

// Yum type represents the resource for package management on
// RHEL and CentOS systems.
//
// Example:
//   pkg = resource.yum.new("emacs")
//   pkg.state = "installed"
type Yum struct {
	BasePackage
}

// NewYum creates a new resource for managing packages
// using the yum package manager on RHEL and CentOS systems
func NewYum(name string) (Resource, error) {
	y := &Yum{
		BasePackage: BasePackage{
			Base: Base{
				Name:              name,
				Type:              "package",
				State:             "installed",
				Require:           make([]string, 0),
				PresentStatesList: []string{"present", "installed"},
				AbsentStatesList:  []string{"absent", "deinstalled"},
				Concurrent:        false,
				Subscribe:         make(TriggerMap),
			},
			Package:       name,
			manager:       "/usr/bin/yum",
			queryArgs:     []string{"-q", "--noplugins", "list", "installed"},
			installArgs:   []string{"--assumeyes", "install"},
			deinstallArgs: []string{"--assumeyes", "remove"},
			updateArgs:    []string{"--assumeyes", "install"},
		},
	}

	return y, nil
}

// PkgNG type represents the resource for package management on
// FreeBSD 9.2+ and DragonflyBSD 4.3+ systems.
//
// Pkg is not automatically bootstrapped - resource will fail,
// if pkg is not installed.
//
// Example:
//   pkg = resource.pkgng.new("mc")
//   pkg.state = "installed"
type PkgNG struct {
	BasePackage
}

// NewPkgNG creates a new resource for managing packages.
func NewPkgNG(name string) (Resource, error) {
	p := &PkgNG{
		BasePackage: BasePackage{
			Base: Base{
				Name:              name,
				Type:              "package",
				State:             "installed",
				Require:           make([]string, 0),
				PresentStatesList: []string{"present", "installed"},
				AbsentStatesList:  []string{"absent", "deinstalled"},
				Concurrent:        false,
				Subscribe:         make(TriggerMap),
			},
			Package:       name,
			manager:       "/usr/local/sbin/pkg",
			queryArgs:     []string{"info", "-e"},
			installArgs:   []string{"install", "-y"},
			deinstallArgs: []string{"remove", "-y"},
			updateArgs:    []string{"upgrade", "-y"},
		},
	}

	return p, nil
}

func init() {
	pkg := ProviderItem{
		Type:      "package",
		Provider:  NewPackage,
		Namespace: DefaultResourceNamespace,
	}
	yum := ProviderItem{
		Type:      "yum",
		Provider:  NewYum,
		Namespace: DefaultResourceNamespace,
	}
	pacman := ProviderItem{
		Type:      "pacman",
		Provider:  NewPacman,
		Namespace: DefaultResourceNamespace,
	}

	pkgng := ProviderItem{
		Type:      "pkgng",
		Provider:  NewPkgNG,
		Namespace: DefaultResourceNamespace,
	}

	RegisterProvider(pkg, yum, pacman, pkgng)
}
