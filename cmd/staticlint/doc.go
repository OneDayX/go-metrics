// Staticlint is a multichecker: one binary that runs many static analyzers
// over Go packages and prints the problems they find. The source code of
// go-metrics must pass it without a single diagnostic.
//
// # Running
//
// Build the binary in the module root and give it the packages to check, the
// same way as to go build or go vet:
//
//	go build -o staticlint ./cmd/staticlint
//	./staticlint ./...
//
// It can also be started without a separate build step, or through make:
//
//	go run ./cmd/staticlint ./...
//	make lint
//
// The same binary works as a vet tool, then go vet loads the packages:
//
//	go vet -vettool=$(pwd)/staticlint ./...
//
// Staticlint loads the packages together with their tests, runs every
// analyzer on them and prints each problem as file:line:column: message.
// The exit code is 0 when nothing is found, 3 when there are diagnostics and
// 1 when the packages cannot be loaded or an analyzer fails.
//
// All analyzers run by default. The flags below change that; staticlint -help
// lists every flag:
//
//	-NAME          run only the analyzers named this way, e.g. -osexit -errcheck
//	-NAME=false    run all analyzers except the named ones
//	-test=false    do not check _test.go files
//	-json          print the diagnostics as JSON
//	-c=N           print N lines of code around each diagnostic
//	-fix           apply the fixes suggested by the analyzers
//
// staticlint help prints the list of analyzers and staticlint help NAME the
// full documentation and flags of one of them.
//
// # Standard analyzers
//
// These come from golang.org/x/tools/go/analysis/passes. It is the go vet set
// plus a few analyzers that go vet does not run:
//
//   - appends: append called with no values to add, like append(s).
//   - asmdecl: assembly files that do not match the Go declarations of their functions.
//   - assign: useless self-assignments like x = x.
//   - atomic: x = atomic.AddInt64(&x, 1), where the assignment breaks atomicity.
//   - atomicalign: 64-bit atomic operations on fields that are not 64-bit aligned on 32-bit platforms.
//   - bools: suspicious boolean expressions, like x == 1 || x == 1 or x != a || x != b.
//   - buildtag: malformed or misplaced //go:build lines.
//   - cgocall: Go pointers passed to C in violation of the cgo rules.
//   - composite: composite literals of struct types from other packages without field names.
//   - copylock: locks, like sync.Mutex or a struct holding one, copied by value.
//   - deepequalerrors: reflect.DeepEqual used to compare errors.
//   - defers: defer f(time.Since(start)), where time.Since runs at once, not at the end.
//   - directive: misplaced or unknown Go toolchain directives, like //go:debug.
//   - errorsas: errors.As called with a second argument that is not a pointer.
//   - framepointer: assembly that overwrites the frame pointer before saving it.
//   - gofixdirective (package gofix): wrong //go:fix inline directives.
//   - hostport: addresses for net.Dial built with fmt.Sprintf("%s:%d"), which break on IPv6; net.JoinHostPort should be used.
//   - httpmux: Go 1.22 http.ServeMux patterns in a module for an older Go version.
//   - httpresponse: an http.Response used before the error of the request is checked.
//   - ifaceassert: interface-to-interface type assertions that can never succeed.
//   - inline: calls to functions marked //go:fix inline, usually deprecated ones, that should be inlined.
//   - loopclosure: loop variables captured by goroutines or deferred closures (matters before Go 1.22).
//   - lostcancel: a cancel function from context.WithCancel or WithTimeout that is not called on every path.
//   - nilfunc: functions compared with nil, which is always true or always false.
//   - nilness: nil pointer dereferences and nil checks that are always true or false.
//   - printf: format strings that do not match the arguments of fmt.Printf and its wrappers.
//   - reflectvaluecompare: reflect.Value compared with == or reflect.DeepEqual.
//   - scannererr: a bufio.Scanner loop without a final check of Scanner.Err.
//   - shift: shifts equal to or longer than the width of the integer.
//   - sigchanyzer: unbuffered channels passed to signal.Notify, which may miss a signal.
//   - slog: log/slog calls with mismatched key-value pairs.
//   - sortslice: sort.Slice called with something that is not a slice.
//   - sqlrowserr: a sql.Rows loop without a final check of Rows.Err.
//   - stdmethods: methods like String, MarshalJSON or WriteTo with a wrong signature.
//   - stdversion: standard library symbols newer than the Go version in go.mod.
//   - stringintconv: string(x) for an integer x, which makes a rune, not a decimal number.
//   - structtag: malformed struct tags and duplicate json or xml field names.
//   - testinggoroutine: t.Fatal and similar called from a goroutine started by a test.
//   - tests: test, benchmark, fuzz and example functions with a wrong name or signature.
//   - timeformat: time layouts with 2006-02-01 (year-day-month), usually meant as 2006-01-02.
//   - unmarshal: json, xml or gob decoding into a value that is not a pointer.
//   - unreachable: code that can never run, like statements after return or panic.
//   - unsafeptr: invalid conversions of uintptr to unsafe.Pointer.
//   - unusedresult: dropped results of functions without side effects, like fmt.Sprintf or errors.New.
//   - unusedwrite: writes to struct fields or array elements that are never read, usually a write to a copy.
//   - waitgroup: sync.WaitGroup.Add called inside the goroutine it waits for.
//
// Some analyzers of the package are not used on purpose. shadow complains
// about the usual redeclaration of err in an inner scope, fieldalignment asks
// to reorder struct fields to save a few bytes, and the modernize suite only
// suggests newer syntax (go fix applies it). inspect, buildssa and ctrlflow
// only prepare data for other analyzers, and findcall, pkgfact and
// usesgenerics are examples that report nothing useful.
//
// # Staticcheck: SA class
//
// All analyzers of the SA class from honnef.co/go/tools/staticcheck. They
// find real bugs and code that does not do what it seems to do.
//
// SA1: misuse of the standard library:
//
//   - SA1000: Invalid regular expression.
//   - SA1001: Invalid template.
//   - SA1002: Invalid format in time.Parse.
//   - SA1003: Unsupported argument to functions in encoding/binary.
//   - SA1004: Suspiciously small untyped constant in time.Sleep.
//   - SA1005: Invalid first argument to exec.Command.
//   - SA1006: Printf with dynamic first argument and no further arguments.
//   - SA1007: Invalid URL in net/url.Parse.
//   - SA1008: Non-canonical key in http.Header map.
//   - SA1010: (*regexp.Regexp).FindAll called with n == 0, which will always return zero results.
//   - SA1011: Various methods in the "strings" package expect valid UTF-8, but invalid input is provided.
//   - SA1012: A nil context.Context is being passed to a function, consider using context.TODO instead.
//   - SA1013: io.Seeker.Seek is being called with the whence constant as the first argument, but it should be the second.
//   - SA1014: Non-pointer value passed to Unmarshal or Decode.
//   - SA1015: Using time.Tick in a way that will leak. Consider using time.NewTicker, and only use time.Tick in tests, commands and endless functions.
//   - SA1016: Trapping a signal that cannot be trapped.
//   - SA1017: Channels used with os/signal.Notify should be buffered.
//   - SA1018: strings.Replace called with n == 0, which does nothing.
//   - SA1019: Using a deprecated function, variable, constant or field.
//   - SA1020: Using an invalid host:port pair with a net.Listen-related function.
//   - SA1021: Using bytes.Equal to compare two net.IP.
//   - SA1023: Modifying the buffer in an io.Writer implementation.
//   - SA1024: A string cutset contains duplicate characters.
//   - SA1025: It is not possible to use (*time.Timer).Reset's return value correctly.
//   - SA1026: Cannot marshal channels or functions.
//   - SA1027: Atomic access to 64-bit variable must be 64-bit aligned.
//   - SA1028: sort.Slice can only be used on slices.
//   - SA1029: Inappropriate key in call to context.WithValue.
//   - SA1030: Invalid argument in call to a strconv function.
//   - SA1031: Overlapping byte slices passed to an encoder.
//   - SA1032: Wrong order of arguments to errors.Is.
//
// SA2: concurrency mistakes:
//
//   - SA2000: (*sync.WaitGroup).Add called inside the goroutine, leading to a race condition.
//   - SA2001: Empty critical section, did you mean to defer the unlock?.
//   - SA2002: Called testing.T.FailNow or SkipNow in a goroutine, which isn't allowed.
//   - SA2003: Deferred Lock right after locking, likely meant to defer Unlock instead.
//
// SA3: testing mistakes:
//
//   - SA3000: TestMain doesn't call os.Exit, hiding test failures.
//   - SA3001: Assigning to b.N in benchmarks distorts the results.
//
// SA4: code that does nothing useful:
//
//   - SA4000: Binary operator has identical expressions on both sides.
//   - SA4001: &*x gets simplified to x, it does not copy x.
//   - SA4003: Comparing unsigned values against negative values is pointless.
//   - SA4004: The loop exits unconditionally after one iteration.
//   - SA4005: Field assignment that will never be observed. Did you mean to use a pointer receiver?.
//   - SA4006: A value assigned to a variable is never read before being overwritten. Forgotten error check or dead code?.
//   - SA4008: The variable in the loop condition never changes, are you incrementing the wrong variable?.
//   - SA4009: A function argument is overwritten before its first use.
//   - SA4010: The result of append will never be observed anywhere.
//   - SA4011: Break statement with no effect. Did you mean to break out of an outer loop?.
//   - SA4012: Comparing a value against NaN even though no value is equal to NaN.
//   - SA4013: Negating a boolean twice (!!b) is the same as writing b. This is either redundant, or a typo.
//   - SA4014: An if/else if chain has repeated conditions and no side-effects; if the condition didnt match the first time, it wont match the second time, either.
//   - SA4015: Calling functions like math.Ceil on floats converted from integers doesn't do anything useful.
//   - SA4016: Certain bitwise operations, such as x ^ 0, do not do anything useful.
//   - SA4017: Discarding the return values of a function without side effects, making the call pointless.
//   - SA4018: Self-assignment of variables.
//   - SA4019: Multiple, identical build constraints in the same file.
//   - SA4020: Unreachable case clause in a type switch.
//   - SA4021: "x = append(y)" is equivalent to "x = y".
//   - SA4022: Comparing the address of a variable against nil.
//   - SA4023: Impossible comparison of interface value with untyped nil.
//   - SA4024: Checking for impossible return value from a builtin function.
//   - SA4025: Integer division of literals that results in zero.
//   - SA4026: Go constants cannot express negative zero.
//   - SA4027: (*net/url.URL).Query returns a copy, modifying it doesn't change the URL.
//   - SA4028: x % 1 is always zero.
//   - SA4029: Ineffective attempt at sorting slice.
//   - SA4030: Ineffective attempt at generating random number.
//   - SA4031: Checking never-nil value against nil.
//   - SA4032: Comparing runtime.GOOS or runtime.GOARCH against impossible value.
//
// SA5: correctness problems:
//
//   - SA5000: Assignment to nil map.
//   - SA5001: Deferring Close before checking for a possible error.
//   - SA5002: The empty for loop ("for {}") spins and can block the scheduler.
//   - SA5003: Defers in infinite loops will never execute.
//   - SA5004: "for { select { .." with an empty default branch spins.
//   - SA5005: The finalizer references the finalized object, preventing garbage collection.
//   - SA5007: Infinite recursive call.
//   - SA5008: Invalid struct tag.
//   - SA5009: Invalid Printf call.
//   - SA5010: Impossible type assertion.
//   - SA5012: Passing odd-sized slice to function expecting even size.
//
// SA6: performance problems:
//
//   - SA6000: Using regexp.Match or related in a loop, should use regexp.Compile.
//   - SA6001: Missing an optimization opportunity when indexing maps by byte slices.
//   - SA6002: Storing non-pointer values in sync.Pool allocates memory.
//   - SA6003: Converting a string to a slice of runes before ranging over it.
//   - SA6005: Inefficient string comparison with strings.ToLower or strings.ToUpper.
//   - SA6006: Using io.WriteString to write []byte.
//
// SA9: dubious code:
//
//   - SA9001: Defers in range loops may not run when you expect them to.
//   - SA9002: Using a non-octal os.FileMode that looks like it was meant to be in octal.
//   - SA9003: Empty body in an if or else branch.
//   - SA9004: Only the first constant has an explicit type.
//   - SA9005: Trying to marshal a struct with no public fields nor custom marshaling.
//   - SA9006: Dubious bit shifting of a fixed size integer value.
//   - SA9007: Deleting a directory that shouldn't be deleted.
//   - SA9008: else branch of a type assertion is probably not reading the right value.
//   - SA9009: Ineffectual Go compiler directive.
//   - SA9010: Returned function should be called in defer.
//
// # Staticcheck: other classes
//
// A few analyzers from the other staticcheck classes: S (honnef.co/go/tools/simple,
// code that can be simpler), ST (honnef.co/go/tools/stylecheck, style rules)
// and QF (honnef.co/go/tools/quickfix, refactorings). They are listed in
// otherChecks in main.go:
//
//   - S1002: comparison with a boolean constant, like if x == true, instead of if x.
//   - S1008: if cond { return true }; return false, instead of return cond.
//   - S1039: fmt.Sprint called with a single string literal, which returns it unchanged.
//   - ST1005: error strings that start with a capital letter or end with punctuation; they are often wrapped into other messages.
//   - ST1016: methods of one type that use different receiver names.
//   - ST1019: the same package imported more than once.
//   - QF1003: an if/else-if chain that compares one variable and can be a tagged switch.
//
// # Public analyzers
//
//   - errcheck (github.com/kisielk/errcheck/errcheck): calls whose error result is not checked.
//     An error assigned to _ counts as ignored on purpose. The -errcheck.blank,
//     -errcheck.assert and -errcheck.exclude flags make the check stricter or softer.
//   - bodyclose (github.com/timakin/bodyclose/passes/bodyclose): an http.Response whose Body is
//     never closed. Such a body keeps its connection busy and leaks memory.
//
// # osexit
//
// Our own analyzer from the package
// github.com/OneDayX/go-metrics/cmd/staticlint/osexit. It reports direct
// calls to os.Exit in the main function of the main package, including calls
// in closures inside main. os.Exit skips the deferred calls, so files,
// database connections and logs are not closed or flushed. Calls in other
// functions and indirect calls, like log.Fatal, are allowed, and generated
// files, like the main package of go test, are skipped.
package main
