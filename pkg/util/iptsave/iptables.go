package iptsave

import (
	"github.com/golang/glog"
	"bufio"
	"io"
	"fmt"
)

// IPtables represents iptables configuration.
type IPtables struct {
	Tables []*IPtable
	currentRule *IPrule
}

// lastTable return pointer to the last IPtable in IPtables.
func (i *IPtables) lastTable() *IPtable {
	glog.V(1).Info("In lastTable()")
	if len(i.Tables) == 0 { return nil }

	t := i.Tables[len(i.Tables)-1]
	glog.V(2).Info("In lastTable returning with ", t.Name)
	return t
}

// IPtable represents tables in iptables.
type IPtable struct {
	Name string
	Chains []*IPchain
}

// lastChain returns pointer to the last IPchain in IPtable.
func (i *IPtable) lastChain() *IPchain {
	glog.V(1).Info("In lastChain()")
	if len(i.Chains) == 0 { return nil }

	c := i.Chains[len(i.Chains)-1]
	return c
}

// chainByName looks for IPchain with corresponding name and returns a pointer to it.
func (i *IPtable) chainByName(name string) *IPchain {
	glog.V(1).Info("In chainByName()")

	for n, c := range i.Chains {
		if c.Name == name {
			ret := i.Chains[n]
			return ret
		}
	}

	return nil
}

// IPchain represents a chain in iptables.
type IPchain struct {
	Name string
	Policy string
	Counters string
	Rules []*IPrule
}

// lastRule returns pointer to the last IPchain in IPtable.
func (i *IPchain) lastRule() *IPrule {
	if len(i.Rules) == 0 { return nil }

	r := i.Rules[len(i.Rules)-1]
	return r
}

// IPrule represents a rule in iptables.
type IPrule struct {
	Match []*Match
	Action  IPtablesAction
}

// Match represents a match in iptables rule.
type Match struct {
	Negated bool
	Body string
}

// IPtablesAction represents an action in iptables rule.
type IPtablesAction struct {
	Type string
	Body string
}

// IPtablesComment represents a comment in iptables.
type IPtablesComment string

// Parse reads iptables configuration from input.
func (i *IPtables) Parse(input io.Reader) {
	bufReader := bufio.NewReader(input)	
	lexer := newLexer(bufReader)
	i.parse(lexer)
}

func (i *IPtables) parse(lexer *Lexer) {
	for {
		item := lexer.NextItem()
		// fmt.Printf("Discovered item of type %s with body %s \n", item.Type, item.Body)
		i.parseItem(item)

		if item.Type == ItemError || item.Type == ItemEOF {
			break
		}
	}
}

func (i *IPtables) parseItem(item Item) {
	switch item.Type {
	case itemComment:
		return
	case itemTable:
		i.Tables = append(i.Tables, &IPtable{Name: item.Body})
	case itemChain:
		table := i.lastTable()
		if table == nil { panic("Chain before table") } // TODO crash here

		glog.V(1).Infof("In ParseItem adding chain %s to the table %s", item.Body, table.Name)

		table.Chains = append(table.Chains, &IPchain{Name: item.Body})
	case itemChainPolicy:
		table := i.lastTable()

		glog.V(2).Infof("In ParseItem table %s has %d chains", table.Name, len(table.Chains))

		chain := table.lastChain()
		if table == nil || chain == nil { panic("Chain policy before table/chain") } // TODO crash here

		chain.Policy = item.Body
	case itemChainCounter:
		table := i.lastTable()
		chain := table.lastChain()
		if table == nil || chain == nil { panic("Chain policy before table/chain") } // TODO crash here

		chain.Counters = item.Body
	case itemCommit:
		return // TODO, ignored for now, should probably be in the model
	case itemRule:
		table := i.lastTable()
		chain := table.chainByName(item.Body)
		if table == nil || chain == nil { panic("Rule before table/chain") } // TODO crash here

		newRule := new(IPrule)
		chain.Rules = append(chain.Rules, newRule)

		i.currentRule = newRule
	case itemRuleMatch:
		if i.currentRule == nil { panic("RuleMatch before table/chain/rule") } // TODO crash here

		i.currentRule.Match = append(i.currentRule.Match, &Match{Body: item.Body})
	case itemAction:
		if i.currentRule == nil { panic("RuleMatch before table/chain/rule") } // TODO crash here
		i.currentRule.Action = IPtablesAction{Body: item.Body}
		
	}

	return
}

// Render produces iptables-restore compatible representation of current structure.
func (i *IPtables) Render() string {
	result := ""

	for _, t := range i.Tables {
		tableRules := ""
		result += fmt.Sprintf("*%s\n", t.Name)

		for _, c := range t.Chains {
			result += fmt.Sprintf(":%s %s %s\n", c.Name, c.Policy, c.Counters)

			for _, r := range c.Rules {
				tableRules += fmt.Sprintf("-A %s ", c.Name)

				for _, m := range r.Match {
					tableRules += fmt.Sprintf("%s ", m.Body)
				}

				tableRules += fmt.Sprintf("-j %s\n", r.Action.Body)
			}
		}

		result += tableRules
		result += fmt.Sprintf("COMMIT\n")
	}

	return result
}
