package dsl

// Script is a parsed fight script.
type Script struct {
	Main    []Stmt
	OnHit   *Block
	OnBlock *Block
	// handler declaration positions (for duplicate detection)
	OnHitPos, OnBlockPos Pos
}

// Block is a braced statement list.
type Block struct {
	LBrace Pos
	Stmts  []Stmt
	// HadErrors is set when statements were dropped during error recovery,
	// so follow-on checks such as "empty loop" are suppressed.
	HadErrors bool
}

// Stmt is a statement node.
type Stmt interface {
	StmtPos() Pos
}

// ActionStmt is an action such as `kick`, `move_forward x3`, `high_block hold 6`.
type ActionStmt struct {
	Pos    Pos
	Action Action
	N      int // count / hold / wait frames; 0 if omitted
	HasN   bool
	NPos   Pos
}

// IfStmt is an if/elif/else chain.
type IfStmt struct {
	Pos      Pos
	Branches []Branch
	Else     *Block
}

// Branch is one `if`/`elif` arm.
type Branch struct {
	Pos  Pos // position of the if/elif keyword
	Cond Expr
	Body *Block
}

// RepeatStmt is `repeat N { ... }`.
type RepeatStmt struct {
	Pos  Pos
	N    int
	NPos Pos
	Body *Block
}

// LoopStmt is `loop { ... }`.
type LoopStmt struct {
	Pos  Pos
	Body *Block
}

func (s *ActionStmt) StmtPos() Pos { return s.Pos }
func (s *IfStmt) StmtPos() Pos     { return s.Pos }
func (s *RepeatStmt) StmtPos() Pos { return s.Pos }
func (s *LoopStmt) StmtPos() Pos   { return s.Pos }

// Expr is a condition expression node.
type Expr interface {
	ExprPos() Pos
}

// LogicOp is `and` / `or`.
type LogicOp int

const (
	OpAnd LogicOp = iota
	OpOr
)

// LogicExpr is `L and R` / `L or R`.
type LogicExpr struct {
	Pos  Pos
	Op   LogicOp
	L, R Expr
}

// NotExpr is `not X`.
type NotExpr struct {
	Pos Pos
	X   Expr
}

// CmpExpr is `L <op> R`.
type CmpExpr struct {
	Pos  Pos
	Op   string // < <= > >= == !=
	L, R Expr
}

// SensorExpr reads a sensor; Arg is used by in_range.
type SensorExpr struct {
	Pos    Pos
	Sensor Sensor
	Arg    Attack
}

// IntLit is an integer literal.
type IntLit struct {
	Pos Pos
	Val int
}

func (e *LogicExpr) ExprPos() Pos  { return e.Pos }
func (e *NotExpr) ExprPos() Pos    { return e.Pos }
func (e *CmpExpr) ExprPos() Pos    { return e.Pos }
func (e *SensorExpr) ExprPos() Pos { return e.Pos }
func (e *IntLit) ExprPos() Pos     { return e.Pos }

// Name returns the display name of a sensor read, e.g. "in_range(sweep)".
func (e *SensorExpr) Name() string {
	if e.Sensor == SenInRange {
		return "in_range(" + e.Arg.String() + ")"
	}
	return e.Sensor.String()
}
