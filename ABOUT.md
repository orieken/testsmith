# About TestSmith

## The Story

TestSmith was built as a learning project by **Oscar Rieken** (<oscar.rieken@gmail.com>) to explore the intersection of static code analysis, test automation, and AI-assisted development.

What started as a simple idea—"what if we could auto-generate test scaffolds from source code?"—evolved into a comprehensive tool that demonstrates:

- **AST Parsing**: Deep dive into Python's Abstract Syntax Tree for code analysis
- **Project Structure Detection**: Building intelligent systems that adapt to any codebase
- **Code Generation**: Creating maintainable, idempotent code generators
- **LLM Integration**: Practical applications of AI in developer tooling
- **CI/CD Automation**: Modern release pipelines with GitHub Actions
- **Binary Distribution**: Cross-platform packaging with PyInstaller

## Built for Fun, Built to Learn

This project was created purely for the joy of learning and building. Every feature—from the core analysis engine to the dependency graph visualization to the watch mode—was an opportunity to explore new concepts and push the boundaries of what's possible with Python tooling.

The goal was never to build a commercial product, but rather to:
- Learn by doing
- Experiment with different architectural patterns
- Explore the practical limits of AST-based analysis
- Have fun solving interesting problems

## Philosophy

TestSmith embodies a few core principles:

1. **Zero Configuration**: Tools should work out of the box
2. **Idempotency**: Safe to run multiple times without breaking things
3. **Project-Agnostic**: Adapt to any codebase structure
4. **Developer-Friendly**: Clear output, sensible defaults, helpful errors

## Technical Highlights

Some interesting technical challenges solved along the way:

- **Import Classification**: Distinguishing stdlib, internal, and external imports without configuration
- **Shared Fixture Strategy**: Generating reusable mocks that stay in sync with dependencies
- **Dependency Graph Analysis**: Building and visualizing module coupling metrics
- **Coverage Gap Prioritization**: Combining coverage status with dependency metrics to suggest what to test next
- **Watch Mode Debouncing**: Handling rapid file changes without overwhelming the system

## Open Source

TestSmith is open source under the MIT License. Feel free to use it, learn from it, fork it, or contribute to it. The code is meant to be read, studied, and improved upon.

If you find it useful or learn something from it, that's wonderful. If you find bugs or have ideas for improvements, contributions are welcome!

---

**Contact**: 
- Personal: <oriekenjr@gmail.com>
- Consulting: <oscar@rieken.consulting>

**Portfolio**: https://rieken-portfolio.netlify.app/

**License**: MIT (see [LICENSE](LICENSE) file)

**Repository**: https://github.com/orieken/testsmith
