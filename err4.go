package main

//go:generate err4gen .

var then = func(err *error, ok func(), catch func()) {
	switch {
	case *err == nil && ok != nil:
		ok()
	case *err != nil && catch != nil:
		catch()
	}
}
