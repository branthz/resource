Vim�UnDo� ��G���4]%r
�I�]e�2�L:�9�l��  e                                   Z*��    _�                    m       ����                                                                                                                                                                                                                                                                                                                                                             Z*��     �  l  m          9			log.Printf("start server:%s==================\n",addr)5�_�                    m       ����                                                                                                                                                                                                                                                                                                                                                             Z*��    �              e   5// Package caddy implements the Caddy server manager.   //   // To use this package:   //   1//   1. Set the AppName and AppVersion variables.   2//   2. Call LoadCaddyfile() to get the Caddyfile.   ://      Pass in the name of the server type (like "http").   7//   3. Call caddy.Start() to start Caddy. You get back   7//      an Instance, on which you can call Restart() to   (//      restart it or Stop() to stop it.   //   6// You should call Wait() on your instance to wait for   1// all servers to quit before your process exits.   package caddy       import (   	"bytes"   	"fmt"   	"io"   	"io/ioutil"   	"log"   	"net"   	"os"   
	"strconv"   
	"strings"   	"sync"   	"time"       #	"github.com/mholt/caddy/caddyfile"   )       &// Configurable application parameters   var (   +	// AppName is the name of the application.   	AppName string       1	// AppVersion is the version of the application.   	AppVersion string       F	// Quiet mode will not show any informative output on initialization.   	Quiet bool       1	// PidFile is the path to the pidfile to create.   	PidFile string       C	// GracefulTimeout is the maximum duration of a graceful shutdown.   	GracefulTimeout time.Duration       1	// isUpgrade will be set to true if this process   5	// was started as part of an upgrade, where a parent   #	// Caddy process started this one.   	isUpgrade bool       .	// started will be set to true when the first   -	// instance is started; it never gets set to   	// false after that.   	started bool       8	// mu protects the variables 'isUpgrade' and 'started'.   	mu sync.Mutex   )       @// Instance contains the state of servers created as a result of   D// calling Start and can be used to access or control those servers.   type Instance struct {   8	// serverType is the name of the instance's server type   	serverType string       H	// caddyfileInput is the input configuration text used for this process   	caddyfileInput Input       3	// wg is used to wait for all servers to shut down   	wg *sync.WaitGroup       5	// context is the context created for this instance.   	context Context       8	// servers is the list of servers with their listeners.   	servers []ServerListener       5	// these callbacks execute when certain events occur   E	onFirstStartup  []func() error // starting, not as part of a restart   F	onStartup       []func() error // starting, even as part of a restart   ;	onRestart       []func() error // before restart commences   F	onShutdown      []func() error // stopping, even as part of a restart   E	onFinalShutdown []func() error // stopping, not as part of a restart   }       ,// Servers returns the ServerListeners in i.   Bfunc (i *Instance) Servers() []ServerListener { return i.servers }       5// Stop stops all servers contained in i. It does NOT   // execute shutdown callbacks.   !func (i *Instance) Stop() error {   	// stop the servers   	for _, s := range i.servers {   .		if gs, ok := s.server.(GracefulServer); ok {   $			if err := gs.Stop(); err != nil {   <				log.Printf("[ERROR] Stopping %s: %v", gs.Address(), err)   			}   		}   	}       E	// splice i out of instance list, causing it to be garbage-collected   	instancesMu.Lock()   "	for j, other := range instances {   		if other == i {   8			instances = append(instances[:j], instances[j+1:]...)   			break   		}   	}   	instancesMu.Unlock()       	return nil   }       >// ShutdownCallbacks executes all the shutdown callbacks of i,   @// including ones that are scheduled only for the final shutdown   >// of i. An error returned from one does not stop execution of   5// the rest. All the non-nil errors will be returned.   0func (i *Instance) ShutdownCallbacks() []error {   	var errs []error   ,	for _, shutdownFunc := range i.onShutdown {   		err := shutdownFunc()   		if err != nil {   			errs = append(errs, err)   		}   	}   6	for _, finalShutdownFunc := range i.onFinalShutdown {   		err := finalShutdownFunc()   		if err != nil {   			errs = append(errs, err)   		}   	}   	return errs   }       B// Restart replaces the servers in i with new servers created from   ?// executing the newCaddyfile. Upon success, it returns the new   ?// instance to replace i. Upon failure, i will not be replaced.   Cfunc (i *Instance) Restart(newCaddyfile Input) (*Instance, error) {    	log.Println("[INFO] Reloading")       	i.wg.Add(1)   	defer i.wg.Done()       	// run restart callbacks   !	for _, fn := range i.onRestart {   		err := fn()   		if err != nil {   			return i, err   		}   	}       	if newCaddyfile == nil {   !		newCaddyfile = i.caddyfileInput   	}       B	// Add file descriptors of all the sockets that are capable of it   -	restartFds := make(map[string]restartTriple)   	for _, s := range i.servers {   (		gs, srvOk := s.server.(GracefulServer)   #		ln, lnOk := s.listener.(Listener)   #		pc, pcOk := s.packet.(PacketConn)   		if srvOk {   			if lnOk && pcOk {   R				restartFds[gs.Address()] = restartTriple{server: gs, listener: ln, packet: pc}   				continue   			}   			if lnOk {   F				restartFds[gs.Address()] = restartTriple{server: gs, listener: ln}   				continue   			}   			if pcOk {   D				restartFds[gs.Address()] = restartTriple{server: gs, packet: pc}   				continue   			}   		}   	}       E	// create new instance; if the restart fails, it is simply discarded   F	newInst := &Instance{serverType: newCaddyfile.ServerType(), wg: i.wg}       !	// attempt to start new instance   ?	err := startWithListenerFds(newCaddyfile, newInst, restartFds)   	if err != nil {   		return i, err   	}       "	// success! stop the old instance   ,	for _, shutdownFunc := range i.onShutdown {   		err := shutdownFunc()   		if err != nil {   			return i, err   		}   	}   		i.Stop()       )	log.Println("[INFO] Reloading complete")       	return newInst, nil   }       :// SaveServer adds s and its associated listener ln to the   7// internally-kept list of servers that is running. For   5// saved servers, graceful restarts will be provided.   :func (i *Instance) SaveServer(s Server, ln net.Listener) {   G	i.servers = append(i.servers, ServerListener{server: s, listener: ln})   }       9// HasListenerWithAddress returns whether this package is   6// tracking a server using a listener with the address   // addr.   /func HasListenerWithAddress(addr string) bool {   	instancesMu.Lock()   	defer instancesMu.Unlock()   !	for _, inst := range instances {   $		for _, sln := range inst.servers {   -			if listenerAddrEqual(sln.listener, addr) {   				return true   			}   		}   	}   	return false   }       7// listenerAddrEqual compares a listener's address with   7// addr. Extra care is taken to match addresses with an   6// empty hostname portion, as listeners tend to report   7// [::]:80, for example, when the matching address that   ,// created the listener might be simply :80.   ;func listenerAddrEqual(ln net.Listener, addr string) bool {   	lnAddr := ln.Addr().String()   /	hostname, port, err := net.SplitHostPort(addr)   	if err != nil {   		return lnAddr == addr   	}   ,	if lnAddr == net.JoinHostPort("::", port) {   		return true   	}   1	if lnAddr == net.JoinHostPort("0.0.0.0", port) {   		return true   	}   (	return hostname != "" && lnAddr == addr   }       =// TCPServer is a type that can listen and serve connections.   E// A TCPServer must associate with exactly zero or one net.Listeners.   type TCPServer interface {   6	// Listen starts listening by creating a new listener   1	// and returning it. It does not start accepting   2	// connections. For UDP-only servers, this method   +	// can be a no-op that returns (nil, nil).   	Listen() (net.Listener, error)       5	// Serve starts serving using the provided listener.   8	// Serve must start the server loop nearly immediately,   7	// or at least not return any errors before the server   7	// loop begins. Serve blocks indefinitely, or in other   4	// words, until the server is stopped. For UDP-only   9	// servers, this method can be a no-op that returns nil.   	Serve(net.Listener) error   }       9// UDPServer is a type that can listen and serve packets.   G// A UDPServer must associate with exactly zero or one net.PacketConns.   type UDPServer interface {   >	// ListenPacket starts listening by creating a new packetconn   >	// and returning it. It does not start accepting connections.   ;	// TCP-only servers may leave this method blank and return   	// (nil, nil).   '	ListenPacket() (net.PacketConn, error)       =	// ServePacket starts serving using the provided packetconn.   >	// ServePacket must start the server loop nearly immediately,   7	// or at least not return any errors before the server   =	// loop begins. ServePacket blocks indefinitely, or in other   =	// words, until the server is stopped. For TCP-only servers,   0	// this method can be a no-op that returns nil.   "	ServePacket(net.PacketConn) error   }       ?// Server is a type that can listen and serve. It supports both   <// TCP and UDP, although the UDPServer interface can be used   // for more than just UDP.   //   D// If the server uses TCP, it should implement TCPServer completely.   =// If it uses UDP or some other protocol, it should implement   C// UDPServer completely. If it uses both, both interfaces should be   A// fully implemented. Any unimplemented methods should be made as   (// no-ops that simply return nil values.   type Server interface {   
	TCPServer   
	UDPServer   }       4// Stopper is a type that can stop serving. The stop   ,// does not necessarily have to be graceful.   type Stopper interface {   .	// Stop stops the server. It blocks until the   !	// server is completely stopped.   	Stop() error   }       7// GracefulServer is a Server and Stopper, the stopping   9// of which is graceful (whatever that means for the kind   :// of server being implemented). It must be able to return   8// the address it is configured to listen on so that its   9// listener can be paired with it upon graceful restarts.   6// The net.Listener that a GracefulServer creates must   6// implement the Listener interface for restarts to be   /// graceful (assuming the listener is for TCP).   type GracefulServer interface {   	Server   	Stopper       1	// Address returns the address the server should   /	// listen on; it is used to pair the server to   0	// its listener during a graceful/zero-downtime   0	// restart. Thus when implementing this method,   -	// you must not access a listener to get the   +	// address; you must store the address the   )	// server is to serve on some other way.   	Address() string   }       A// Listener is a net.Listener with an underlying file descriptor.   ?// A server's listener should implement this interface if it is   $// to support zero-downtime reloads.   type Listener interface {   	net.Listener   	File() (*os.File, error)   }       E// PacketConn is a net.PacketConn with an underlying file descriptor.   A// A server's packetconn should implement this interface if it is   J// to support zero-downtime reloads (in sofar this holds true for datagram   // connections).   type PacketConn interface {   	net.PacketConn   	File() (*os.File, error)   }       7// AfterStartup is an interface that can be implemented   9// by a server type that wants to run some code after all   .// servers for the same Instance have started.   type AfterStartup interface {   	OnStartupComplete()   }       <// LoadCaddyfile loads a Caddyfile by calling the plugged in   >// Caddyfile loader methods. An error is returned if more than   >// one loader returns a non-nil Caddyfile input. If no loaders   >// load a Caddyfile, the default loader is used. If no default   <// loader is registered or it returns nil, the server type's   ;// default Caddyfile is loaded. If the server type does not   ?// specify any default Caddyfile value, then an empty Caddyfile   ?// is returned. Consequently, this function never returns a nil   (// value as long as there are no errors.   6func LoadCaddyfile(serverType string) (Input, error) {   *	// Ask plugged-in loaders for a Caddyfile   /	cdyfile, err := loadCaddyfileInput(serverType)   	if err != nil {   		return nil, err   	}       	// Otherwise revert to default   	if cdyfile == nil {   $		cdyfile = DefaultInput(serverType)   	}       	// Still nil? Geez.   	if cdyfile == nil {   6		cdyfile = CaddyfileInput{ServerTypeName: serverType}   	}       	return cdyfile, nil   }       5// Wait blocks until all of i's servers have stopped.   func (i *Instance) Wait() {   	i.wg.Wait()   }       =// CaddyfileFromPipe loads the Caddyfile input from f if f is   >// not interactive input. f is assumed to be a pipe or stream,   =// such as os.Stdin. If f is not a pipe, no error is returned   =// but the Input value will be nil. An error is only returned   =// if there was an error reading the pipe, even if the length   // of what was read is 0.   Ffunc CaddyfileFromPipe(f *os.File, serverType string) (Input, error) {   	fi, err := f.Stat()   4	if err == nil && fi.Mode()&os.ModeCharDevice == 0 {   8		// Note that a non-nil error is not a problem. Windows   7		// will not create a stdin if there is no pipe, which   9		// produces an error when calling Stat(). But Unix will   9		// make one either way, which is why we also check that   		// bitmask.   `		// NOTE: Reading from stdin after this fails (e.g. for the let's encrypt email address) (OS X)   $		confBody, err := ioutil.ReadAll(f)   		if err != nil {   			return nil, err   		}   		return CaddyfileInput{   			Contents:       confBody,   			Filepath:       f.Name(),   			ServerTypeName: serverType,   		}, nil   	}       :	// not having input from the pipe is not itself an error,   "	// just means no input to return.   	return nil, nil   }       4// Caddyfile returns the Caddyfile used to create i.   &func (i *Instance) Caddyfile() Input {   	return i.caddyfileInput   }       /// Start starts Caddy with the given Caddyfile.   //   <// This function blocks until all the servers are listening.   .func Start(cdyfile Input) (*Instance, error) {   	writePidFile()   M	inst := &Instance{serverType: cdyfile.ServerType(), wg: new(sync.WaitGroup)}   6	return inst, startWithListenerFds(cdyfile, inst, nil)   }       efunc startWithListenerFds(cdyfile Input, inst *Instance, restartFds map[string]restartTriple) error {   	if cdyfile == nil {   		cdyfile = CaddyfileInput{}   	}       :	err := ValidateAndExecuteDirectives(cdyfile, inst, false)   	if err != nil {   		return err   	}       )	slist, err := inst.context.MakeServers()   	if err != nil {   		return err   	}       	// run startup callbacks   	if restartFds == nil {   8		for _, firstStartupFunc := range inst.onFirstStartup {   			err := firstStartupFunc()   			if err != nil {   				return err   			}   		}   	}   -	for _, startupFunc := range inst.onStartup {   		err := startupFunc()   		if err != nil {   			return err   		}   	}       ,	err = startServers(slist, inst, restartFds)   	if err != nil {   		return err   	}       	instancesMu.Lock()   $	instances = append(instances, inst)   	instancesMu.Unlock()       1	// run any AfterStartup callbacks if this is not   7	// part of a restart; then show file descriptor notice   	if restartFds == nil {   &		for _, srvln := range inst.servers {   2			if srv, ok := srvln.server.(AfterStartup); ok {   				srv.OnStartupComplete()   			}   		}   		if !Quiet {   '			for _, srvln := range inst.servers {   4				if !IsLoopback(srvln.listener.Addr().String()) {   					checkFdlimit()   
					break   				}   			}   		}   	}       
	mu.Lock()   	started = true   	mu.Unlock()       	return nil   }       H// ValidateAndExecuteDirectives will load the server blocks from cdyfile   H// by parsing it, then execute the directives configured by it and store   H// the resulting server blocks into inst. If justValidate is true, parse   G// callbacks will not be executed between directives, since the purpose   /// is only to check the input for valid syntax.   [func ValidateAndExecuteDirectives(cdyfile Input, inst *Instance, justValidate bool) error {       U	// If parsing only inst will be nil, create an instance for this function call only.   	if justValidate {   M		inst = &Instance{serverType: cdyfile.ServerType(), wg: new(sync.WaitGroup)}   	}       "	stypeName := cdyfile.ServerType()       '	stype, err := getServerType(stypeName)   	if err != nil {   		return err   	}       	inst.caddyfileInput = cdyfile       ]	sblocks, err := loadServerBlocks(stypeName, cdyfile.Path(), bytes.NewReader(cdyfile.Body()))   	if err != nil {   		return err   	}       "	inst.context = stype.NewContext()   	if inst.context == nil {   G		return fmt.Errorf("server type %s produced a nil Context", stypeName)   	}       I	sblocks, err = inst.context.InspectServerBlocks(cdyfile.Path(), sblocks)   	if err != nil {   		return err   	}       Y	err = executeDirectives(inst, cdyfile.Path(), stype.Directives(), sblocks, justValidate)   	if err != nil {   		return err   	}       	return nil       }       7func executeDirectives(inst *Instance, filename string,   Q	directives []string, sblocks []caddyfile.ServerBlock, justValidate bool) error {   @	// map of server block ID to map of directive name to whatever.   1	storages := make(map[int]map[string]interface{})       C	// It is crucial that directives are executed in the proper order.   ?	// We loop with the directives on the outer loop so we execute   I	// a directive for all server blocks before going to the next directive.   B	// This is important mainly due to the parsing callbacks (below).   !	for _, dir := range directives {   		for i, sb := range sblocks {   			var once sync.Once   !			if _, ok := storages[i]; !ok {   .				storages[i] = make(map[string]interface{})   			}        			for j, key := range sb.Keys {   5				// Execute directive if it is in the server block   )				if tokens, ok := sb.Tokens[dir]; ok {   					controller := &Controller{   						instance:  inst,   						Key:       key,   @						Dispenser: caddyfile.NewDispenserTokens(filename, tokens),   6						OncePerServerBlock: func(f func() error) error {   							var err error   							once.Do(func() {   								err = f()   								})   							return err   						},   						ServerBlockIndex:    i,   						ServerBlockKeyIndex: j,   #						ServerBlockKeys:     sb.Keys,   ,						ServerBlockStorage:  storages[i][dir],   					}       8					setup, err := DirectiveAction(inst.serverType, dir)   					if err != nil {   						return err   					}       					err = setup(controller)   					if err != nil {   						return err   					}       V					storages[i][dir] = controller.ServerBlockStorage // persist for this server block   				}   			}   		}       		if !justValidate {   D			// See if there are any callbacks to execute after this directive   A			if allCallbacks, ok := parsingCallbacks[inst.serverType]; ok {   "				callbacks := allCallbacks[dir]   (				for _, callback := range callbacks {   3					if err := callback(inst.context); err != nil {   						return err   					}   				}   			}   		}   	}       	return nil   }       cfunc startServers(serverList []Server, inst *Instance, restartFds map[string]restartTriple) error {   -	errChan := make(chan error, len(serverList))       	for _, s := range serverList {   		var (   			ln  net.Listener   			pc  net.PacketConn   			err error   		)       3		// If this is a reload and s is a GracefulServer,   /		// reuse the listener for a graceful restart.   <		if gs, ok := s.(GracefulServer); ok && restartFds != nil {   			addr := gs.Address()   '			if old, ok := restartFds[addr]; ok {   				// listener   				if old.listener != nil {   %					file, err := old.listener.File()   					if err != nil {   						return err   					}   %					ln, err = net.FileListener(file)   					if err != nil {   						return err   					}   					file.Close()   				}   				// packetconn   				if old.packet != nil {   #					file, err := old.packet.File()   					if err != nil {   						return err   					}   '					pc, err = net.FilePacketConn(file)   					if err != nil {   						return err   					}   					file.Close()   				}   			}   		}       		if ln == nil {   			ln, err = s.Listen()   			if err != nil {   				return err   			}   		}   		if pc == nil {   			pc, err = s.ListenPacket()   			if err != nil {   				return err   			}   		}       		inst.wg.Add(2)   I		go func(s Server, ln net.Listener, pc net.PacketConn, inst *Instance) {   			defer inst.wg.Done()       			go func() {   				errChan <- s.Serve(ln)   				defer inst.wg.Done()   			}()   			errChan <- s.ServePacket(pc)   		}(s, ln, pc, inst)       Z		inst.servers = append(inst.servers, ServerListener{server: s, listener: ln, packet: pc})   	}       7	// Log errors that may be returned from Serve() calls,   =	// these errors should only be occurring in the server loop.   	go func() {   		for err := range errChan {   			if err == nil {   				continue   			}   I			if strings.Contains(err.Error(), "use of closed network connection") {   5				// this error is normal when closing the listener   				continue   			}   			log.Println(err)   		}   	}()       	return nil   }       ;func getServerType(serverType string) (ServerType, error) {   %	stype, ok := serverTypes[serverType]   	if ok {   		return stype, nil   	}   	if len(serverTypes) == 0 {   ?		return ServerType{}, fmt.Errorf("no server types plugged in")   	}   	if serverType == "" {   		if len(serverTypes) == 1 {   &			for _, stype := range serverTypes {   				return stype, nil   			}   		}   U		return ServerType{}, fmt.Errorf("multiple server types available; must choose one")   	}   H	return ServerType{}, fmt.Errorf("unknown server type '%s'", serverType)   }       ffunc loadServerBlocks(serverType, filename string, input io.Reader) ([]caddyfile.ServerBlock, error) {   /	validDirectives := ValidDirectives(serverType)   G	serverBlocks, err := caddyfile.Parse(filename, input, validDirectives)   	if err != nil {   		return nil, err   	}   K	if len(serverBlocks) == 0 && serverTypes[serverType].DefaultInput != nil {   4		newInput := serverTypes[serverType].DefaultInput()   6		serverBlocks, err = caddyfile.Parse(newInput.Path(),   5			bytes.NewReader(newInput.Body()), validDirectives)   		if err != nil {   			return nil, err   		}   	}   	return serverBlocks, nil   }       @// Stop stops ALL servers. It blocks until they are all stopped.   =// It does NOT execute shutdown callbacks, and it deletes all   ;// instances after stopping is completed. Do not re-use any   2// references to old instances after calling Stop.   func Stop() error {   6	// This awkward for loop is to avoid a deadlock since   3	// inst.Stop() also acquires the instancesMu lock.   	for {   		instancesMu.Lock()   		if len(instances) == 0 {   			break   		}   		inst := instances[0]   		instancesMu.Unlock()   %		if err := inst.Stop(); err != nil {   >			log.Printf("[ERROR] Stopping %s: %v", inst.serverType, err)   		}   	}   	return nil   }       8// IsLoopback returns true if the hostname of addr looks   :// explicitly like a common local hostname. addr must only   (// be a host or a host:port combination.   #func IsLoopback(addr string) bool {   (	host, _, err := net.SplitHostPort(addr)   	if err != nil {   7		host = addr // happens if the addr is just a hostname   	}   	return host == "localhost" ||   &		strings.Trim(host, "[]") == "::1" ||   !		strings.HasPrefix(host, "127.")   }       <// Upgrade re-launches the process, preserving the listeners   >// for a graceful restart. It does NOT load new configuration;   7// it only starts the process anew with a fresh binary.   //   $// TODO: This is not yet implemented   func Upgrade() error {   %	return fmt.Errorf("not implemented")   1	// TODO: have child process set isUpgrade = true   }       ?// IsUpgrade returns true if this process is part of an upgrade   ;// where a parent caddy process spawned this one to upgrade   // the binary.   func IsUpgrade() bool {   
	mu.Lock()   	defer mu.Unlock()   	return isUpgrade   }       9// Started returns true if at least one instance has been   8// started by this package. It never gets reset to false   // once it is set to true.   func Started() bool {   
	mu.Lock()   	defer mu.Unlock()   	return started   }       1// CaddyfileInput represents a Caddyfile as input   .// and is simply a convenient way to implement   // the Input interface.   type CaddyfileInput struct {   	Filepath       string   	Contents       []byte   	ServerTypeName string   }       // Body returns c.Contents.   ;func (c CaddyfileInput) Body() []byte { return c.Contents }       // Path returns c.Filepath.   ;func (c CaddyfileInput) Path() string { return c.Filepath }       #// ServerType returns c.ServerType.   Gfunc (c CaddyfileInput) ServerType() string { return c.ServerTypeName }       ;// Input represents a Caddyfile; its contents and file path   ?// (which should include the file name at the end of the path).   8// If path does not apply (e.g. piped input) you may use   A// any understandable value. The path is mainly used for logging,   !// error messages, and debugging.   type Input interface {   	// Gets the Caddyfile contents   	Body() []byte       $	// Gets the path to the origin file   	Path() string       1	// The type of server this input is intended for   	ServerType() string   }       3// DefaultInput returns the default Caddyfile input   0// to use when it is otherwise empty or missing.   0// It uses the default host and port (depends on   3// host, e.g. localhost is 2015, otherwise 443) and   // root.   ,func DefaultInput(serverType string) Input {   +	if _, ok := serverTypes[serverType]; !ok {   		return nil   	}   1	if serverTypes[serverType].DefaultInput == nil {   		return nil   	}   .	return serverTypes[serverType].DefaultInput()   }       =// writePidFile writes the process ID to the file at PidFile.   )// It does nothing if PidFile is not set.   func writePidFile() error {   	if PidFile == "" {   		return nil   	}   0	pid := []byte(strconv.Itoa(os.Getpid()) + "\n")   ,	return ioutil.WriteFile(PidFile, pid, 0644)   }       type restartTriple struct {   	server   GracefulServer   	listener Listener   	packet   PacketConn   }       var (   /	// instances is the list of running Instances.   	instances []*Instance       #	// instancesMu protects instances.   	instancesMu sync.Mutex   )       var (   J	// DefaultConfigFile is the name of the configuration file that is loaded   -	// by default if no other file is specified.    	DefaultConfigFile = "Caddyfile"   )5��