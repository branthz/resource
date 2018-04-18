Vim�UnDo� 6�?�6�P�V�u(��U�s�s���>��ԅEY  !                                  Z���    _�                     T       ����                                                                                                                                                                                                                                                                                                                                                             Z��v     �   T   V  !      		�   T   V       5�_�                    U       ����                                                                                                                                                                                                                                                                                                                                                             Z�ǂ     �   T   V  !      		5�_�                    U       ����                                                                                                                                                                                                                                                                                                                                                             Z�ǆ     �   T   V  !      		fmt.Printf()	5�_�                    U       ����                                                                                                                                                                                                                                                                                                                                                             Z�Ǉ     �   T   V  !      		fmt.Printf("")	5�_�                    U        ����                                                                                                                                                                                                                                                                                                                                                             Z�ǐ     �   T   V  !      "		fmt.Printf("pid filepath:%s\n")	5�_�                    U        ����                                                                                                                                                                                                                                                                                                                                                             Z�ǘ    �              !   @// Package coremain contains the functions for starting CoreDNS.   package coremain       import (   		"errors"   	"flag"   	"fmt"   	"io/ioutil"   	"log"   	"os"   
	"runtime"   
	"strconv"   
	"strings"   
	"syscall"       	"github.com/mholt/caddy"       ,	"github.com/coredns/coredns/core/dnsserver"       	// Plug in CoreDNS   $	_ "github.com/coredns/coredns/core"   )       func init() {   d	// Reset flag.CommandLine to get rid of unwanted flags for instance from glog (used in kubernetes).   '	// And readd the once we want to keep.   #	flag.VisitAll(func(f *flag.Flag) {   *		if _, ok := flagsBlacklist[f.Name]; ok {   				return   		}   &		flagsToKeep = append(flagsToKeep, f)   	})       A	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)    	for _, f := range flagsToKeep {   $		flag.Var(f.Value, f.Name, f.Usage)   	}       	caddy.TrapSignals()   %	caddy.DefaultConfigFile = "Corefile"   7	caddy.Quiet = true // don't show init stuff from caddy   	setVersion()       `	flag.StringVar(&conf, "conf", "", "Corefile to load (default \""+caddy.DefaultConfigFile+"\")")   /	flag.StringVar(&cpu, "cpu", "100%", "CPU cap")   C	flag.BoolVar(&plugins, "plugins", false, "List installed plugins")   8	flag.StringVar(&logfile, "log", "", "Process log file")   H	flag.StringVar(&caddy.PidFile, "pidfile", "", "Path to write pid file")   9	flag.BoolVar(&version, "version", false, "Show version")   X	flag.BoolVar(&dnsserver.Quiet, "quiet", false, "Quiet mode (no initialization output)")   H	flag.BoolVar(&reload, "reload", false, "Reload CoreDNS configurations")       D	caddy.RegisterCaddyfileLoader("flag", caddy.LoaderFunc(confLoader))   L	caddy.SetDefaultCaddyfileLoader("default", caddy.LoaderFunc(defaultLoader))   }       $// Run is CoreDNS's main() function.   func Run() {       	flag.Parse()       	caddy.AppName = coreName   	caddy.AppVersion = coreVersion       2	// Set up process log before anything bad happens   	switch logfile {   	case "stdout":   		log.SetOutput(os.Stdout)   	case "stderr":   		log.SetOutput(os.Stderr)   		default:   		log.SetOutput(os.Stdout)   	}   	log.SetFlags(log.LstdFlags)       	if version {   		showVersion()   		os.Exit(0)   	}   	if plugins {   &		fmt.Println(caddy.DescribePlugins())   		os.Exit(0)   	}   	if reload {   0		fmt.Printf("pid filepath:%s\n",caddy.PidFile)	   -		data, err := ioutil.ReadFile(caddy.PidFile)   		if err != nil {   =			fmt.Println("read pid file", caddy.PidFile, "error:", err)   			os.Exit(-1)   		}   <		pid, err := strconv.Atoi(strings.Trim(string(data), "\n"))   		if err != nil {   )			fmt.Println("pid file fmt error", err)   			os.Exit(-1)   		}       $		fmt.Println("Reload CoreDNS", pid)   		p := &os.Process{Pid: pid}   3		if err := p.Signal(syscall.SIGUSR1); err != nil {   3			fmt.Println("sending reload signal failed", err)   		}   		os.Exit(0)   	}   	if caddy.PidFile != "" {   3		if _, err := os.Stat(caddy.PidFile); err == nil {   '			fmt.Println("CoreDNS is running...")   			os.Exit(-1)   		}   	}       	// Set CPU cap   $	if err := setCPU(cpu); err != nil {   		mustLogFatal(err)   	}       	// Get Corefile input   1	corefile, err := caddy.LoadCaddyfile(serverType)   	if err != nil {   		mustLogFatal(err)   	}       	// Start your engines   '	instance, err := caddy.Start(corefile)   	if err != nil {   		mustLogFatal(err)   	}       	logVersion()   	if !dnsserver.Quiet {   		showVersion()   	}       	// Twiddle your thumbs   	instance.Wait()   }       ;// mustLogFatal wraps log.Fatal() in a way that ensures the   <// output is always printed to stderr so the user can see it   >// if the user is still there, even if the process log was not   @// enabled. If this process is an upgrade, however, and the user   <// might not be there anymore, this just logs to the process   // log and exits.   (func mustLogFatal(args ...interface{}) {   	if !caddy.IsUpgrade() {   		log.SetOutput(os.Stderr)   	}   	log.Fatal(args...)   }       7// confLoader loads the Caddyfile using the -conf flag.   9func confLoader(serverType string) (caddy.Input, error) {   	if conf == "" {   		return nil, nil   	}       	if conf == "stdin" {   1		return caddy.CaddyfileFromPipe(os.Stdin, "dns")   	}       '	contents, err := ioutil.ReadFile(conf)   	if err != nil {   		return nil, err   	}   	return caddy.CaddyfileInput{   		Contents:       contents,   		Filepath:       conf,   		ServerTypeName: serverType,   	}, nil   }       G// defaultLoader loads the Corefile from the current working directory.   <func defaultLoader(serverType string) (caddy.Input, error) {   :	contents, err := ioutil.ReadFile(caddy.DefaultConfigFile)   	if err != nil {   		if os.IsNotExist(err) {   			return nil, nil   		}   		return nil, err   	}   	return caddy.CaddyfileInput{   		Contents:       contents,   *		Filepath:       caddy.DefaultConfigFile,   		ServerTypeName: serverType,   	}, nil   }       0// logVersion logs the version that is starting.   <func logVersion() { log.Print("[INFO] " + versionString()) }       3// showVersion prints the version that is starting.   func showVersion() {   	fmt.Print(versionString())   $	if devBuild && gitShortStat != "" {   8		fmt.Printf("%s\n%s\n", gitShortStat, gitFilesModified)   	}   }       9// versionString returns the CoreDNS version as a string.   func versionString() string {   ?	return fmt.Sprintf("%s-%s\n", caddy.AppName, caddy.AppVersion)   }       1// setVersion figures out the version information   &// based on variables set by -ldflags.   func setVersion() {   M	// A development build is one that's not at a tag or has uncommitted changes   .	devBuild = gitTag == "" || gitShortStat != ""       0	// Only set the appVersion if -ldflags was used   )	if gitNearestTag != "" || gitTag != "" {   &		if devBuild && gitNearestTag != "" {   *			appVersion = fmt.Sprintf("%s (+%s %s)",   A				strings.TrimPrefix(gitNearestTag, "v"), gitCommit, buildDate)   		} else if gitTag != "" {   /			appVersion = strings.TrimPrefix(gitTag, "v")   		}   	}   }       /// setCPU parses string cpu and sets GOMAXPROCS   ,// according to its value. It accepts either   -// a number (e.g. 3) or a percent (e.g. 50%).   func setCPU(cpu string) error {   	var numCPU int       	availCPU := runtime.NumCPU()       !	if strings.HasSuffix(cpu, "%") {   		// Percent   		var percent float32   		pctStr := cpu[:len(cpu)-1]   %		pctInt, err := strconv.Atoi(pctStr)   /		if err != nil || pctInt < 1 || pctInt > 100 {   K			return errors.New("invalid CPU value: percentage must be between 1-100")   		}   !		percent = float32(pctInt) / 100   +		numCPU = int(float32(availCPU) * percent)   		} else {   		// Number   		num, err := strconv.Atoi(cpu)   		if err != nil || num < 1 {   U			return errors.New("invalid CPU value: provide a number or percent greater than 0")   		}   		numCPU = num   	}       	if numCPU > availCPU {   		numCPU = availCPU   	}       	runtime.GOMAXPROCS(numCPU)   	return nil   }       -// Flags that control program flow or startup   var (   	conf    string   	cpu     string   	logfile string   	version bool   	plugins bool   	reload  bool   )       7// Build information obtained with the help of -ldflags   var (   <	appVersion = "(untracked dev build)" // inferred at startup   <	devBuild   = true                    // inferred at startup       #	buildDate        string // date -u   H	gitTag           string // git describe --exact-match HEAD 2> /dev/null   ?	gitNearestTag    string // git describe --abbrev=0 --tags HEAD   .	gitCommit        string // git rev-parse HEAD   6	gitShortStat     string // git diff-index --shortstat   ;	gitFilesModified string // git diff-index --name-only HEAD   )       B// flagsBlacklist removes flags with these names from our flagset.   %var flagsBlacklist = map[string]bool{   	"logtostderr":      true,   	"alsologtostderr":  true,   	"v":                true,   	"stderrthreshold":  true,   	"vmodule":          true,   	"log_backtrace_at": true,   	"log_dir":          true,   }       var flagsToKeep []*flag.Flag5�_�                    U   ,    ����                                                                                                                                                                                                                                                                                                                                                             Z�Ǳ    �              !   @// Package coremain contains the functions for starting CoreDNS.   package coremain       import (   		"errors"   	"flag"   	"fmt"   	"io/ioutil"   	"log"   	"os"   
	"runtime"   
	"strconv"   
	"strings"   
	"syscall"       	"github.com/mholt/caddy"       ,	"github.com/coredns/coredns/core/dnsserver"       	// Plug in CoreDNS   $	_ "github.com/coredns/coredns/core"   )       func init() {   d	// Reset flag.CommandLine to get rid of unwanted flags for instance from glog (used in kubernetes).   '	// And readd the once we want to keep.   #	flag.VisitAll(func(f *flag.Flag) {   *		if _, ok := flagsBlacklist[f.Name]; ok {   				return   		}   &		flagsToKeep = append(flagsToKeep, f)   	})       A	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)    	for _, f := range flagsToKeep {   $		flag.Var(f.Value, f.Name, f.Usage)   	}       	caddy.TrapSignals()   %	caddy.DefaultConfigFile = "Corefile"   7	caddy.Quiet = true // don't show init stuff from caddy   	setVersion()       `	flag.StringVar(&conf, "conf", "", "Corefile to load (default \""+caddy.DefaultConfigFile+"\")")   /	flag.StringVar(&cpu, "cpu", "100%", "CPU cap")   C	flag.BoolVar(&plugins, "plugins", false, "List installed plugins")   8	flag.StringVar(&logfile, "log", "", "Process log file")   H	flag.StringVar(&caddy.PidFile, "pidfile", "", "Path to write pid file")   9	flag.BoolVar(&version, "version", false, "Show version")   X	flag.BoolVar(&dnsserver.Quiet, "quiet", false, "Quiet mode (no initialization output)")   H	flag.BoolVar(&reload, "reload", false, "Reload CoreDNS configurations")       D	caddy.RegisterCaddyfileLoader("flag", caddy.LoaderFunc(confLoader))   L	caddy.SetDefaultCaddyfileLoader("default", caddy.LoaderFunc(defaultLoader))   }       $// Run is CoreDNS's main() function.   func Run() {       	flag.Parse()       	caddy.AppName = coreName   	caddy.AppVersion = coreVersion       2	// Set up process log before anything bad happens   	switch logfile {   	case "stdout":   		log.SetOutput(os.Stdout)   	case "stderr":   		log.SetOutput(os.Stderr)   		default:   		log.SetOutput(os.Stdout)   	}   	log.SetFlags(log.LstdFlags)       	if version {   		showVersion()   		os.Exit(0)   	}   	if plugins {   &		fmt.Println(caddy.DescribePlugins())   		os.Exit(0)   	}   	if reload {   0		fmt.Printf("pid filepath:%s\n", caddy.PidFile)   -		data, err := ioutil.ReadFile(caddy.PidFile)   		if err != nil {   =			fmt.Println("read pid file", caddy.PidFile, "error:", err)   			os.Exit(-1)   		}   <		pid, err := strconv.Atoi(strings.Trim(string(data), "\n"))   		if err != nil {   )			fmt.Println("pid file fmt error", err)   			os.Exit(-1)   		}       $		fmt.Println("Reload CoreDNS", pid)   		p := &os.Process{Pid: pid}   3		if err := p.Signal(syscall.SIGUSR1); err != nil {   3			fmt.Println("sending reload signal failed", err)   		}   		os.Exit(0)   	}   	if caddy.PidFile != "" {   3		if _, err := os.Stat(caddy.PidFile); err == nil {   '			fmt.Println("CoreDNS is running...")   			os.Exit(-1)   		}   	}       	// Set CPU cap   $	if err := setCPU(cpu); err != nil {   		mustLogFatal(err)   	}       	// Get Corefile input   1	corefile, err := caddy.LoadCaddyfile(serverType)   	if err != nil {   		mustLogFatal(err)   	}       	// Start your engines   '	instance, err := caddy.Start(corefile)   	if err != nil {   		mustLogFatal(err)   	}       	logVersion()   	if !dnsserver.Quiet {   		showVersion()   	}       	// Twiddle your thumbs   	instance.Wait()   }       ;// mustLogFatal wraps log.Fatal() in a way that ensures the   <// output is always printed to stderr so the user can see it   >// if the user is still there, even if the process log was not   @// enabled. If this process is an upgrade, however, and the user   <// might not be there anymore, this just logs to the process   // log and exits.   (func mustLogFatal(args ...interface{}) {   	if !caddy.IsUpgrade() {   		log.SetOutput(os.Stderr)   	}   	log.Fatal(args...)   }       7// confLoader loads the Caddyfile using the -conf flag.   9func confLoader(serverType string) (caddy.Input, error) {   	if conf == "" {   		return nil, nil   	}       	if conf == "stdin" {   1		return caddy.CaddyfileFromPipe(os.Stdin, "dns")   	}       '	contents, err := ioutil.ReadFile(conf)   	if err != nil {   		return nil, err   	}   	return caddy.CaddyfileInput{   		Contents:       contents,   		Filepath:       conf,   		ServerTypeName: serverType,   	}, nil   }       G// defaultLoader loads the Corefile from the current working directory.   <func defaultLoader(serverType string) (caddy.Input, error) {   :	contents, err := ioutil.ReadFile(caddy.DefaultConfigFile)   	if err != nil {   		if os.IsNotExist(err) {   			return nil, nil   		}   		return nil, err   	}   	return caddy.CaddyfileInput{   		Contents:       contents,   *		Filepath:       caddy.DefaultConfigFile,   		ServerTypeName: serverType,   	}, nil   }       0// logVersion logs the version that is starting.   <func logVersion() { log.Print("[INFO] " + versionString()) }       3// showVersion prints the version that is starting.   func showVersion() {   	fmt.Print(versionString())   $	if devBuild && gitShortStat != "" {   8		fmt.Printf("%s\n%s\n", gitShortStat, gitFilesModified)   	}   }       9// versionString returns the CoreDNS version as a string.   func versionString() string {   ?	return fmt.Sprintf("%s-%s\n", caddy.AppName, caddy.AppVersion)   }       1// setVersion figures out the version information   &// based on variables set by -ldflags.   func setVersion() {   M	// A development build is one that's not at a tag or has uncommitted changes   .	devBuild = gitTag == "" || gitShortStat != ""       0	// Only set the appVersion if -ldflags was used   )	if gitNearestTag != "" || gitTag != "" {   &		if devBuild && gitNearestTag != "" {   *			appVersion = fmt.Sprintf("%s (+%s %s)",   A				strings.TrimPrefix(gitNearestTag, "v"), gitCommit, buildDate)   		} else if gitTag != "" {   /			appVersion = strings.TrimPrefix(gitTag, "v")   		}   	}   }       /// setCPU parses string cpu and sets GOMAXPROCS   ,// according to its value. It accepts either   -// a number (e.g. 3) or a percent (e.g. 50%).   func setCPU(cpu string) error {   	var numCPU int       	availCPU := runtime.NumCPU()       !	if strings.HasSuffix(cpu, "%") {   		// Percent   		var percent float32   		pctStr := cpu[:len(cpu)-1]   %		pctInt, err := strconv.Atoi(pctStr)   /		if err != nil || pctInt < 1 || pctInt > 100 {   K			return errors.New("invalid CPU value: percentage must be between 1-100")   		}   !		percent = float32(pctInt) / 100   +		numCPU = int(float32(availCPU) * percent)   		} else {   		// Number   		num, err := strconv.Atoi(cpu)   		if err != nil || num < 1 {   U			return errors.New("invalid CPU value: provide a number or percent greater than 0")   		}   		numCPU = num   	}       	if numCPU > availCPU {   		numCPU = availCPU   	}       	runtime.GOMAXPROCS(numCPU)   	return nil   }       -// Flags that control program flow or startup   var (   	conf    string   	cpu     string   	logfile string   	version bool   	plugins bool   	reload  bool   )       7// Build information obtained with the help of -ldflags   var (   <	appVersion = "(untracked dev build)" // inferred at startup   <	devBuild   = true                    // inferred at startup       #	buildDate        string // date -u   H	gitTag           string // git describe --exact-match HEAD 2> /dev/null   ?	gitNearestTag    string // git describe --abbrev=0 --tags HEAD   .	gitCommit        string // git rev-parse HEAD   6	gitShortStat     string // git diff-index --shortstat   ;	gitFilesModified string // git diff-index --name-only HEAD   )       B// flagsBlacklist removes flags with these names from our flagset.   %var flagsBlacklist = map[string]bool{   	"logtostderr":      true,   	"alsologtostderr":  true,   	"v":                true,   	"stderrthreshold":  true,   	"vmodule":          true,   	"log_backtrace_at": true,   	"log_dir":          true,   }       var flagsToKeep []*flag.Flag5�_�                     U       ����                                                                                                                                                                                                                                                                                                                                                             Z���    �              !   @// Package coremain contains the functions for starting CoreDNS.   package coremain       import (   		"errors"   	"flag"   	"fmt"   	"io/ioutil"   	"log"   	"os"   
	"runtime"   
	"strconv"   
	"strings"   
	"syscall"       	"github.com/mholt/caddy"       ,	"github.com/coredns/coredns/core/dnsserver"       	// Plug in CoreDNS   $	_ "github.com/coredns/coredns/core"   )       func init() {   d	// Reset flag.CommandLine to get rid of unwanted flags for instance from glog (used in kubernetes).   '	// And readd the once we want to keep.   #	flag.VisitAll(func(f *flag.Flag) {   *		if _, ok := flagsBlacklist[f.Name]; ok {   				return   		}   &		flagsToKeep = append(flagsToKeep, f)   	})       A	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)    	for _, f := range flagsToKeep {   $		flag.Var(f.Value, f.Name, f.Usage)   	}       	caddy.TrapSignals()   %	caddy.DefaultConfigFile = "Corefile"   7	caddy.Quiet = true // don't show init stuff from caddy   	setVersion()       `	flag.StringVar(&conf, "conf", "", "Corefile to load (default \""+caddy.DefaultConfigFile+"\")")   /	flag.StringVar(&cpu, "cpu", "100%", "CPU cap")   C	flag.BoolVar(&plugins, "plugins", false, "List installed plugins")   8	flag.StringVar(&logfile, "log", "", "Process log file")   H	flag.StringVar(&caddy.PidFile, "pidfile", "", "Path to write pid file")   9	flag.BoolVar(&version, "version", false, "Show version")   X	flag.BoolVar(&dnsserver.Quiet, "quiet", false, "Quiet mode (no initialization output)")   H	flag.BoolVar(&reload, "reload", false, "Reload CoreDNS configurations")       D	caddy.RegisterCaddyfileLoader("flag", caddy.LoaderFunc(confLoader))   L	caddy.SetDefaultCaddyfileLoader("default", caddy.LoaderFunc(defaultLoader))   }       $// Run is CoreDNS's main() function.   func Run() {       	flag.Parse()       	caddy.AppName = coreName   	caddy.AppVersion = coreVersion       2	// Set up process log before anything bad happens   	switch logfile {   	case "stdout":   		log.SetOutput(os.Stdout)   	case "stderr":   		log.SetOutput(os.Stderr)   		default:   		log.SetOutput(os.Stdout)   	}   	log.SetFlags(log.LstdFlags)       	if version {   		showVersion()   		os.Exit(0)   	}   	if plugins {   &		fmt.Println(caddy.DescribePlugins())   		os.Exit(0)   	}   	if reload {   0		fmt.Printf("pid filepath:%s\n", caddy.PidFile)   -		data, err := ioutil.ReadFile(caddy.PidFile)   		if err != nil {   =			fmt.Println("read pid file", caddy.PidFile, "error:", err)   			os.Exit(-1)   		}   <		pid, err := strconv.Atoi(strings.Trim(string(data), "\n"))   		if err != nil {   )			fmt.Println("pid file fmt error", err)   			os.Exit(-1)   		}       $		fmt.Println("Reload CoreDNS", pid)   		p := &os.Process{Pid: pid}   3		if err := p.Signal(syscall.SIGUSR1); err != nil {   3			fmt.Println("sending reload signal failed", err)   		}   		os.Exit(0)   	}   	if caddy.PidFile != "" {   3		if _, err := os.Stat(caddy.PidFile); err == nil {   '			fmt.Println("CoreDNS is running...")   			os.Exit(-1)   		}   	}       	// Set CPU cap   $	if err := setCPU(cpu); err != nil {   		mustLogFatal(err)   	}       	// Get Corefile input   1	corefile, err := caddy.LoadCaddyfile(serverType)   	if err != nil {   		mustLogFatal(err)   	}       	// Start your engines   '	instance, err := caddy.Start(corefile)   	if err != nil {   		mustLogFatal(err)   	}       	logVersion()   	if !dnsserver.Quiet {   		showVersion()   	}       	// Twiddle your thumbs   	instance.Wait()   }       ;// mustLogFatal wraps log.Fatal() in a way that ensures the   <// output is always printed to stderr so the user can see it   >// if the user is still there, even if the process log was not   @// enabled. If this process is an upgrade, however, and the user   <// might not be there anymore, this just logs to the process   // log and exits.   (func mustLogFatal(args ...interface{}) {   	if !caddy.IsUpgrade() {   		log.SetOutput(os.Stderr)   	}   	log.Fatal(args...)   }       7// confLoader loads the Caddyfile using the -conf flag.   9func confLoader(serverType string) (caddy.Input, error) {   	if conf == "" {   		return nil, nil   	}       	if conf == "stdin" {   1		return caddy.CaddyfileFromPipe(os.Stdin, "dns")   	}       '	contents, err := ioutil.ReadFile(conf)   	if err != nil {   		return nil, err   	}   	return caddy.CaddyfileInput{   		Contents:       contents,   		Filepath:       conf,   		ServerTypeName: serverType,   	}, nil   }       G// defaultLoader loads the Corefile from the current working directory.   <func defaultLoader(serverType string) (caddy.Input, error) {   :	contents, err := ioutil.ReadFile(caddy.DefaultConfigFile)   	if err != nil {   		if os.IsNotExist(err) {   			return nil, nil   		}   		return nil, err   	}   	return caddy.CaddyfileInput{   		Contents:       contents,   *		Filepath:       caddy.DefaultConfigFile,   		ServerTypeName: serverType,   	}, nil   }       0// logVersion logs the version that is starting.   <func logVersion() { log.Print("[INFO] " + versionString()) }       3// showVersion prints the version that is starting.   func showVersion() {   	fmt.Print(versionString())   $	if devBuild && gitShortStat != "" {   8		fmt.Printf("%s\n%s\n", gitShortStat, gitFilesModified)   	}   }       9// versionString returns the CoreDNS version as a string.   func versionString() string {   ?	return fmt.Sprintf("%s-%s\n", caddy.AppName, caddy.AppVersion)   }       1// setVersion figures out the version information   &// based on variables set by -ldflags.   func setVersion() {   M	// A development build is one that's not at a tag or has uncommitted changes   .	devBuild = gitTag == "" || gitShortStat != ""       0	// Only set the appVersion if -ldflags was used   )	if gitNearestTag != "" || gitTag != "" {   &		if devBuild && gitNearestTag != "" {   *			appVersion = fmt.Sprintf("%s (+%s %s)",   A				strings.TrimPrefix(gitNearestTag, "v"), gitCommit, buildDate)   		} else if gitTag != "" {   /			appVersion = strings.TrimPrefix(gitTag, "v")   		}   	}   }       /// setCPU parses string cpu and sets GOMAXPROCS   ,// according to its value. It accepts either   -// a number (e.g. 3) or a percent (e.g. 50%).   func setCPU(cpu string) error {   	var numCPU int       	availCPU := runtime.NumCPU()       !	if strings.HasSuffix(cpu, "%") {   		// Percent   		var percent float32   		pctStr := cpu[:len(cpu)-1]   %		pctInt, err := strconv.Atoi(pctStr)   /		if err != nil || pctInt < 1 || pctInt > 100 {   K			return errors.New("invalid CPU value: percentage must be between 1-100")   		}   !		percent = float32(pctInt) / 100   +		numCPU = int(float32(availCPU) * percent)   		} else {   		// Number   		num, err := strconv.Atoi(cpu)   		if err != nil || num < 1 {   U			return errors.New("invalid CPU value: provide a number or percent greater than 0")   		}   		numCPU = num   	}       	if numCPU > availCPU {   		numCPU = availCPU   	}       	runtime.GOMAXPROCS(numCPU)   	return nil   }       -// Flags that control program flow or startup   var (   	conf    string   	cpu     string   	logfile string   	version bool   	plugins bool   	reload  bool   )       7// Build information obtained with the help of -ldflags   var (   <	appVersion = "(untracked dev build)" // inferred at startup   <	devBuild   = true                    // inferred at startup       #	buildDate        string // date -u   H	gitTag           string // git describe --exact-match HEAD 2> /dev/null   ?	gitNearestTag    string // git describe --abbrev=0 --tags HEAD   .	gitCommit        string // git rev-parse HEAD   6	gitShortStat     string // git diff-index --shortstat   ;	gitFilesModified string // git diff-index --name-only HEAD   )       B// flagsBlacklist removes flags with these names from our flagset.   %var flagsBlacklist = map[string]bool{   	"logtostderr":      true,   	"alsologtostderr":  true,   	"v":                true,   	"stderrthreshold":  true,   	"vmodule":          true,   	"log_backtrace_at": true,   	"log_dir":          true,   }       var flagsToKeep []*flag.Flag5��