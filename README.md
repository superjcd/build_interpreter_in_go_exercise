# 摘要


## 理解Pratt Parsing（或者说Top Down Operator Precedence）

首先运行`git checkout ch02_06`, 然后查看`parse`文件夹下面的`parse.go`,
涉及到算法的核心的代码主要有以下几段：

### 最上层的parseStatement
```go
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()  
	case token.RETURN:
		return p.parseReturnStatement()
	default:
		return p.parseExpressionStatement()
	}
}
```
parseStatement会在token不是`LET`以及`RETURN`的情况下去调用`parseExpressionStatement`


### parseExpressionStatement

```go
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}
```

`parseExpressionStatement`构建一个`ExpressionStatement`的结构体， 然后紧接着调用`parseExpression`

### parseExpression

这个函数是理解Pratt Parsing的关键！
```go
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()  // 第一步的时候是1， 然后变成（1+2）

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]  // 处理加号；

		if infix == nil {
			return leftExp    //  处理1
		}

		p.nextToken()   // 关键点， curentToken -> peek, peekToken -> nextPeekToken
		leftExp = infix(leftExp)    // 这里会recursice, 第一次会返回 （1 + 2）； 那么在下一次与逆行的时候left 就是（1 + 2）
	}

	return leftExp
}
```
### 例子: 解析1 + 2 +3
关键的代码就是上面这三段(其实主要是**parseExpression**)，我们以解析`1+2+3`为例， 一步一步来分析`parseExpression`究竟做了什么， 首先需要说明的是:我们的目标是把`1 + 2 + 3`解析成一个AST对象（本质是一个中缀表达式）， 它的`LEFT`是`1+2`, 它的`RIGHT`是`3`， 换言之就是把`1+2+3`变成`(1+2)+3`。
我们进入到`parseExpression`
```go
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()  // 第一步的时候是1， 然后变成（1+2）
```
第一个部分会根据当前toekn的类型获取注册的**前缀函数**， 那么哪些token有**前缀函数**呢？

- INT类型， 比如说数值1
- BANG类型， 也就是用来去否的感叹号`!`
- MINUS类型， 就是`-`, 需要注意的是MINUS既有绑定了**前缀函数**， 又绑定了**中缀函数**
- IDENT类型， 也就是一般的符号， 比如`a`啊`b`啊这种

还是回到表达式`1 + 2 + 3`, 我们的curToken(当前token)是`1`，它是一个`Int`类型， 所以它有一个前缀表达式， 我们先不管Int类型的前缀表达式是啥， 反正不是nil， 所以就会来到 leftExp := prefix()

接着会进入到下面一个关键的`foor loop`：

```go
for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence():
```
`!p.peekTokenIs(token.SEMICOLON)`会判断下一个token是不是`;`， 所以只要不到语句结尾它就会是`true`， 所以目前上面的语句可以近似理解为

```go
for true && precedence < p.peekPrecedence():
```
` precedence < p.peekPrecedence()`中的precedence是`parseExpression`的最开始的入参(`parseExpression`的上一层调用者`parseExpressionStatemn`传进来的)， 总之它的precedence是`LOWEST`， 顾名思义就是优先级最低， 所以它自然会小于下个token的precedence(也就是`+`的优先级), 既然说到了`precednece`优先级， 那么我们来看一下有哪些优先级：

```go
const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
)

var precedences = map[token.TokenType]int{
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.SLASH:    PRODUCT,
	token.ASTERISK: PRODUCT,
}
```
可以看到`token.PLUS`(也就是`+`)确实要大于`LOWEST`。 那么就会进入上面的foor loop：

```go
	infix := p.infixParseFns[p.peekToken.Type]  // 处理第一个加号；

		if infix == nil {
			return leftExp    
		}

		p.nextToken()  
		leftExp = infix(leftExp)  
```
在foor loop里面， 我们会先拿到当前token的下一个token(peekToken是获取下一token)的**中缀函数**， 也就是第一个`+`的中缀函数, 然后解析器调用`nexttoken`向前移动一个空格， 然后当前的token就变成了`+`。  
接着通过`leftExp = infix(leftExp)`我们更新了最左表达式leftExp！ 我们进入`+`的**中缀函数**来探究一下具体是怎么更新的： 

``` go
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,  // 1
	}
	precedence := p.curPrecedence() // 这里非常关键， 会记录最近一次infixExpreesion的优先级
	p.nextToken()  // 又一次前进了
	expression.Right = p.parseExpression(precedence) //parseInfixExpression 和 parseExpression互相调用
	return expression
}
```
我一步一步来看， 首先前几行代码就是构造一个AST结构体。接着是很重要的一步， 这里通过`precedence := p.curPrecedence()`记录了当前操作符的优先级(也就是`+`的优先级)， 然后调用
`p.nextToken`让当前token更新为2。注意!**当前token目前是2**。  
然后激动人心的时候到了， 在**中缀函数**里面， 我们再次调用**parseExpression**, 还记得我们是怎样进入到中缀函数( parseInfixExpression)里面的吗--通过**parseExpression**, 现在**中缀函数**又调用了**parseExpression**, 所以神一般的递归又出现了!

接着我们要故地重游， 再次执行`parseExpression`， 只不过现在的precendence不再是LOWEST了， 而是`+`的优先级了, 首先我们还是获取2的**前缀解析函数**（和1是同类型， 你可以简单的理解为就是返回2本身）。然后再一次需要进入到那个该死的foor loop!但是这次我们还要进入吗？
我们再来看看到这个foor loop：
```go 
for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence()
```
一样， 由于没有到结尾(或者说下一个token不是分号)，所以上面的表达式可以简化成：
```go
for  true && precedence < p.peekPrecedence()
```
然后是比较`precednece`（第一个加号）和下一个token(第二个`+`)的优先级， 由于下一个token的优先级也是`+`的优先级， 所以这个foor loop我们是不用进入的！所以直接返回前面`leftExp := prefix() `的`leftExp`(也就是数值2)。
这样我们可以退回到第一个中缀表达式的里面， 把2的值赋予中缀表达式的`RIGHT`， 然后整个表达式(就是之前构造的ast.InfixExpression节点)就被返回。
接着， 我们优惠回到了调用`中缀函数`parseInfixExpression的最外层的parseExpression的foor loop中，  由于最外层的parseExpression的precedence是LOWEST, 所以会再一次地进入for loop。

```go
		infix := p.infixParseFns[p.peekToken.Type]  // 处理第二个加号；

		if infix == nil {
			return leftExp    //  处理1
		}

		p.nextToken()   // 关键点， curent -> peek, peek -> next
		leftExp = infix(leftExp)
```

第二次进入foor loop的时候 我们先调用`p.nextToken`， 那么我们当前的token是第二个`+`, 接着我们的中缀函数将会更新为第二个`+`的中缀函数， 只不过这个时候的leftExp变成了由`1 + 2`构成的AST节点。

然后， 接着第二次进入中缀表达式解析函数`parseInfixExpression`, 用 `1 + 2`构成的这个AST节点会成为新的中缀表达式的`Left`， 然后调用`p.nextToken`, 这个时候当前的token变成了3。

后面的故事就是： 由于3的下一个节点是分号， 所以不进入foor loop， 然后直接返回3作为新构建的`中缀表达式`的RIGHT， 然后接着返回上层。就这么简单！


## 总结
要理解Pratt Parser的关键是要理解那个foor loop, 每一次进入foor loop都会比较最近操作符的优先级和下一个操作符优先级的大小， 只有当当前操作符优先级小于下一个优先级的时候， 才会进一步递归。
假设表达式是1 + 2 * 3, parseInfixExpression调用的parseExpression的foor loop就会进入， 这样最终的结果就会变成1 + (2 * 3) 而不是 （1 + 2）+ 3 。   
另外Pratt Parser构建的AST树会是一颗二叉树。