// Package ipset provides bindings for linux userspace ipset utility
// http://ipset.netfilter.org/ipset.man.html
//
// Ipset allows for managing iptables rules in complex environments where
// otherwise iptables rules would become too huge ot would have to be updated
// too often.
//
// Similarly, this package provides bindings to configure ipset
// programmatically.
//
// Because ipset is typically used in environment with large ipset configurations
// it is not practical ro rely on simple command lines like `ipset add`
// or `ipset create` since thousands of `create` calls would result
// in thousands of forks.
//
// Instead this package utilizes interactive mode provided by `ipset -`
// to execute bulks  of create/delete/add/flush/swap calls in one session.
// Internal object to start and control interactive session called `Handle`
// which implements `io.Writer` and writes directly into ipset stdin.
//
// However, some commands still make more sense in more sense executed one by one
// like `test`, for that reason this package also provides a set of functions
// called `oneshots` (Add/Delete/etc...)
//
// Since ipset can export it's configuration as xml this package provides structures to
// ipset xml structures and constructors for these structures.
//
// Logging: this package is mostly silent to avoid messing with ipset stderr
// but some debug loggin can be enabled using RLOG_TRACE_LEVEL=3 environment variable.
//
// Typical session starts as
//
//	iset, _ := ipset.Load()
//	for _, set := range iset.Sets {
//		fmt.Printf("Set %s of type %s has %d members\n", set.Name, set.Type, len(set.Members))
//	}
//
//	Output:
//	Set host of type hash:net has 2 members
//	Set host2 of type hash:net has 12 members
//	Set timeoutSet of type hash:ip has 0 members
//	Set commentSet of type hash:ip has 1 members
//	Set countersSet of type hash:ip has 1 members
//	Set skbSet of type hash:ip has 1 members
//	Set host3 of type hash:net has 1 members
//	Set super of type list:set has 2 members
//
//
// Interactive sessions workflow
// Pros: useful to create/delete large sets
// Cons: no error handling
//
//	1. Acquire the handle.
//	handle, _ := ipset.NewHandle(HandleWithBin("/my/ipset/binary"))
//
//	2. Start the session.
//	_ = handle.Start()
//
//	3. Call Add/Delete/etc methods of handle.
//	newSet, _ = ipset.NewSet("mynewset", SetHashNetIface, SetWithComment())
//	_ = handle.Create(newSet)
//
//	4. When you done shut down the session.
//	_ = handle.Quit()
//
//	5. And cleanup the resources.
//	ctx, cacnel := context.WithTimeout(...)
//	_ = handle.Wait(ctx)
//
//	that's it.
//
// And non-interactive session might be useful for commands that make sense
// Pros: clear error and output
// Cons: fork per call
//
//	# ipset save
//	Output:
//	create super list:set size 8
//	add super host
//
//	testSet, _ = ipset.NewSet("super", SetListSet)
//	testMember, _ = ipset.NewMember("host", newSet)
//	_, err := ipset.Test(testSet)
package ipset
