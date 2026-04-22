type Context = Record<string, unknown>;

enum TokenType {
  Number,
  String,
  Identifier,
  Operator,
  LeftParen,
  RightParen,
  Dot,
  EOF,
}

interface Token {
  type: TokenType;
  value: string;
}

function tokenize(expr: string): Token[] {
  const tokens: Token[] = [];
  let i = 0;
  while (i < expr.length) {
    const ch = expr[i];
    if (/\s/.test(ch)) { i++; continue; }
    if (ch === '(') { tokens.push({ type: TokenType.LeftParen, value: '(' }); i++; continue; }
    if (ch === ')') { tokens.push({ type: TokenType.RightParen, value: ')' }); i++; continue; }
    if (ch === '.') { tokens.push({ type: TokenType.Dot, value: '.' }); i++; continue; }

    if (ch === "'" || ch === '"') {
      const quote = ch;
      let str = '';
      i++;
      while (i < expr.length && expr[i] !== quote) { str += expr[i]; i++; }
      i++;
      tokens.push({ type: TokenType.String, value: str });
      continue;
    }

    if (/\d/.test(ch)) {
      let num = '';
      while (i < expr.length && /[\d.]/.test(expr[i])) { num += expr[i]; i++; }
      tokens.push({ type: TokenType.Number, value: num });
      continue;
    }

    const twoChar = expr.slice(i, i + 2);
    if (['==', '!=', '>=', '<=', '&&', '||'].includes(twoChar)) {
      tokens.push({ type: TokenType.Operator, value: twoChar });
      i += 2;
      continue;
    }

    if (['+', '-', '*', '/', '>', '<', '!'].includes(ch)) {
      tokens.push({ type: TokenType.Operator, value: ch });
      i++;
      continue;
    }

    if (/[a-zA-Z_]/.test(ch)) {
      let id = '';
      while (i < expr.length && /[a-zA-Z0-9_]/.test(expr[i])) { id += expr[i]; i++; }
      if (id === 'true') { tokens.push({ type: TokenType.Number, value: '1' }); }
      else if (id === 'false') { tokens.push({ type: TokenType.Number, value: '0' }); }
      else { tokens.push({ type: TokenType.Identifier, value: id }); }
      continue;
    }

    i++;
  }
  tokens.push({ type: TokenType.EOF, value: '' });
  return tokens;
}

class Parser {
  private tokens: Token[];
  private pos: number = 0;
  private ctx: Context;

  constructor(tokens: Token[], ctx: Context) {
    this.tokens = tokens;
    this.ctx = ctx;
  }

  private peek(): Token { return this.tokens[this.pos]; }
  private advance(): Token { return this.tokens[this.pos++]; }

  private match(type: TokenType, value?: string): boolean {
    const t = this.peek();
    if (t.type === type && (value === undefined || t.value === value)) {
      this.advance();
      return true;
    }
    return false;
  }

  parse(): unknown {
    const result = this.parseOr();
    return result;
  }

  private parseOr(): unknown {
    let left = this.parseAnd();
    while (this.peek().type === TokenType.Operator && this.peek().value === '||') {
      this.advance();
      const right = this.parseAnd();
      left = !!(left) || !!(right);
    }
    return left;
  }

  private parseAnd(): unknown {
    let left = this.parseEquality();
    while (this.peek().type === TokenType.Operator && this.peek().value === '&&') {
      this.advance();
      const right = this.parseEquality();
      left = !!(left) && !!(right);
    }
    return left;
  }

  private parseEquality(): unknown {
    let left = this.parseComparison();
    while (this.peek().type === TokenType.Operator && (this.peek().value === '==' || this.peek().value === '!=')) {
      const op = this.advance().value;
      const right = this.parseComparison();
      if (op === '==') left = left === right;
      else left = left !== right;
    }
    return left;
  }

  private parseComparison(): unknown {
    let left = this.parseAddSub();
    while (this.peek().type === TokenType.Operator && ['>', '<', '>=', '<='].includes(this.peek().value)) {
      const op = this.advance().value;
      const right = this.parseAddSub();
      const l = Number(left), r = Number(right);
      if (op === '>') left = l > r;
      else if (op === '<') left = l < r;
      else if (op === '>=') left = l >= r;
      else left = l <= r;
    }
    return left;
  }

  private parseAddSub(): unknown {
    let left = this.parseMulDiv();
    while (this.peek().type === TokenType.Operator && (this.peek().value === '+' || this.peek().value === '-')) {
      const op = this.advance().value;
      const right = this.parseMulDiv();
      if (op === '+') left = Number(left) + Number(right);
      else left = Number(left) - Number(right);
    }
    return left;
  }

  private parseMulDiv(): unknown {
    let left = this.parseUnary();
    while (this.peek().type === TokenType.Operator && (this.peek().value === '*' || this.peek().value === '/')) {
      const op = this.advance().value;
      const right = this.parseUnary();
      if (op === '*') left = Number(left) * Number(right);
      else {
        const divisor = Number(right);
        left = divisor === 0 ? 0 : Number(left) / divisor;
      }
    }
    return left;
  }

  private parseUnary(): unknown {
    if (this.peek().type === TokenType.Operator && this.peek().value === '!') {
      this.advance();
      return !this.parseUnary();
    }
    if (this.peek().type === TokenType.Operator && this.peek().value === '-') {
      this.advance();
      return -Number(this.parsePrimary());
    }
    return this.parsePrimary();
  }

  private parsePrimary(): unknown {
    const token = this.peek();

    if (token.type === TokenType.Number) {
      this.advance();
      return parseFloat(token.value);
    }

    if (token.type === TokenType.String) {
      this.advance();
      return token.value;
    }

    if (token.type === TokenType.LeftParen) {
      this.advance();
      const result = this.parseOr();
      this.match(TokenType.RightParen);
      return result;
    }

    if (token.type === TokenType.Identifier) {
      return this.parseIdentifier();
    }

    this.advance();
    return undefined;
  }

  private parseIdentifier(): unknown {
    const name = this.advance().value;

    if (this.peek().type === TokenType.LeftParen) {
      return this.parseFunction(name);
    }

    let value: unknown = this.ctx[name];

    while (this.peek().type === TokenType.Dot) {
      this.advance();
      if (this.peek().type === TokenType.Identifier) {
        const prop = this.advance().value;
        if (value !== null && value !== undefined && typeof value === 'object') {
          value = (value as Record<string, unknown>)[prop];
        } else {
          return undefined;
        }
      }
    }

    return value;
  }

  private parseFunction(name: string): unknown {
    this.advance(); // consume (
    const args: unknown[] = [];
    while (this.peek().type !== TokenType.RightParen && this.peek().type !== TokenType.EOF) {
      args.push(this.parseOr());
    }
    this.match(TokenType.RightParen);

    const nums = args.filter((a): a is number => typeof a === 'number');
    switch (name) {
      case 'sum': return nums.reduce((a, b) => a + b, 0);
      case 'count': return nums.length;
      case 'avg': return nums.length > 0 ? nums.reduce((a, b) => a + b, 0) / nums.length : 0;
      case 'min': return nums.length > 0 ? Math.min(...nums) : 0;
      case 'max': return nums.length > 0 ? Math.max(...nums) : 0;
      default: return undefined;
    }
  }
}

export function evaluate(expr: string, ctx: Context): unknown {
  if (!expr || expr.trim() === '') return undefined;
  try {
    const tokens = tokenize(expr);
    const parser = new Parser(tokens, ctx);
    return parser.parse();
  } catch {
    return undefined;
  }
}
