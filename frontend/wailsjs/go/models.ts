export namespace main {
	
	export class CalculatedColumn {
	    Name: string;
	    Formula: string;
	    Percent: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CalculatedColumn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Formula = source["Formula"];
	        this.Percent = source["Percent"];
	    }
	}
	export class ColumnDTO {
	    id: string;
	    title: string;
	    source: string;
	    type: string;
	    visible: boolean;
	    highlight_sign: boolean;
	    background: string;
	    align: string;
	
	    static createFrom(source: any = {}) {
	        return new ColumnDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.source = source["source"];
	        this.type = source["type"];
	        this.visible = source["visible"];
	        this.highlight_sign = source["highlight_sign"];
	        this.background = source["background"];
	        this.align = source["align"];
	    }
	}
	export class SubtotalRow {
	    group_value: string;
	    values: Record<string, string>;
	    total: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SubtotalRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.group_value = source["group_value"];
	        this.values = source["values"];
	        this.total = source["total"];
	    }
	}
	export class DatasetDTO {
	    columns: ColumnDTO[];
	    rows: any[];
	    total_rows: number;
	    duplicated: number;
	    csv_rows: number;
	    enriched: number;
	    source_files: string[];
	    subtotals: SubtotalRow[];
	
	    static createFrom(source: any = {}) {
	        return new DatasetDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = this.convertValues(source["columns"], ColumnDTO);
	        this.rows = source["rows"];
	        this.total_rows = source["total_rows"];
	        this.duplicated = source["duplicated"];
	        this.csv_rows = source["csv_rows"];
	        this.enriched = source["enriched"];
	        this.source_files = source["source_files"];
	        this.subtotals = this.convertValues(source["subtotals"], SubtotalRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DatasetSettings {
	    decimals: number;
	    font_size: number;
	    row_height: number;
	    column_widths: Record<string, number>;
	    so_column: number;
	    join_excel_column: string;
	    formula_title: string;
	    formula: string;
	    subtotal_column: string;
	    subtotal_enabled: boolean;
	    column_titles: Record<string, string>;
	    column_order: string[];
	    max_columns: number;
	    visible_columns: string[];
	    column_decimals: Record<string, number>;
	    subtotal_columns: string[];
	    column_percent: Record<string, boolean>;
	    column_currency: Record<string, boolean>;
	    column_thousands: Record<string, boolean>;
	    highlight_negative: Record<string, boolean>;
	    column_highlight_sign: Record<string, boolean>;
	    column_background: Record<string, string>;
	    column_align: Record<string, string>;
	    column_types: Record<string, string>;
	    calculated_columns: CalculatedColumn[];
	    subtotal_agg: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new DatasetSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.decimals = source["decimals"];
	        this.font_size = source["font_size"];
	        this.row_height = source["row_height"];
	        this.column_widths = source["column_widths"];
	        this.so_column = source["so_column"];
	        this.join_excel_column = source["join_excel_column"];
	        this.formula_title = source["formula_title"];
	        this.formula = source["formula"];
	        this.subtotal_column = source["subtotal_column"];
	        this.subtotal_enabled = source["subtotal_enabled"];
	        this.column_titles = source["column_titles"];
	        this.column_order = source["column_order"];
	        this.max_columns = source["max_columns"];
	        this.visible_columns = source["visible_columns"];
	        this.column_decimals = source["column_decimals"];
	        this.subtotal_columns = source["subtotal_columns"];
	        this.column_percent = source["column_percent"];
	        this.column_currency = source["column_currency"];
	        this.column_thousands = source["column_thousands"];
	        this.highlight_negative = source["highlight_negative"];
	        this.column_highlight_sign = source["column_highlight_sign"];
	        this.column_background = source["column_background"];
	        this.column_align = source["column_align"];
	        this.column_types = source["column_types"];
	        this.calculated_columns = this.convertValues(source["calculated_columns"], CalculatedColumn);
	        this.subtotal_agg = source["subtotal_agg"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

